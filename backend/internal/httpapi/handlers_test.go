package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"calculator/internal/calculator"
)

func newTestRouter(opts Options) http.Handler {
	return NewRouter(calculator.Default(), opts)
}

func do(t *testing.T, h http.Handler, method, target, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func readError(t *testing.T, rec *httptest.ResponseRecorder) APIError {
	t.Helper()
	var env struct{ Error APIError }
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not an error envelope: %v (%s)", err, rec.Body)
	}
	return env.Error
}

func TestCalculateSuccess(t *testing.T) {
	h := newTestRouter(Options{})
	cases := []struct {
		body string
		want float64
	}{
		{`{"operation":"add","operands":[2,3]}`, 5},
		{`{"operation":"subtract","operands":[2,3]}`, -1},
		{`{"operation":"multiply","operands":[-4,2.5]}`, -10},
		{`{"operation":"divide","operands":[10,4]}`, 2.5},
		{`{"operation":"power","operands":[2,8]}`, 256},
		{`{"operation":"sqrt","operands":[81]}`, 9},
		{`{"operation":"percent","operands":[200,15]}`, 30},
		{`{"operation":"add","operands":[0,0]}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/calculate", "application/json; charset=utf-8", tc.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q", ct)
			}
			var got calculateResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Result != tc.want {
				t.Errorf("result = %v, want %v", got.Result, tc.want)
			}
		})
	}
}

func TestCalculateErrors(t *testing.T) {
	h := newTestRouter(Options{})
	cases := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
	}{
		{"divide by zero", "application/json", `{"operation":"divide","operands":[1,0]}`, 422, CodeDivisionByZero},
		{"sqrt negative", "application/json", `{"operation":"sqrt","operands":[-4]}`, 422, CodeDomainError},
		{"overflow", "application/json", `{"operation":"power","operands":[10,400]}`, 422, CodeNonFiniteResult},
		{"unknown operation", "application/json", `{"operation":"modulo","operands":[1,2]}`, 400, CodeUnsupportedOperation},
		{"missing operation", "application/json", `{"operands":[1,2]}`, 400, CodeUnsupportedOperation},
		{"empty operation", "application/json", `{"operation":"","operands":[1,2]}`, 400, CodeUnsupportedOperation},
		{"missing operands", "application/json", `{"operation":"add"}`, 400, CodeInvalidOperands},
		{"too few operands", "application/json", `{"operation":"add","operands":[1]}`, 400, CodeInvalidOperands},
		{"too many operands", "application/json", `{"operation":"sqrt","operands":[1,2]}`, 400, CodeInvalidOperands},
		{"null operand", "application/json", `{"operation":"add","operands":[1,null]}`, 400, CodeInvalidOperands},
		{"string operand", "application/json", `{"operation":"add","operands":[1,"2"]}`, 400, CodeInvalidOperands},
		{"out of range operand", "application/json", `{"operation":"add","operands":[1e400,1]}`, 400, CodeInvalidOperands},
		{"operands not array", "application/json", `{"operation":"add","operands":5}`, 400, CodeInvalidOperands},
		{"operation wrong type", "application/json", `{"operation":5,"operands":[1,2]}`, 400, CodeInvalidJSON},
		{"malformed json", "application/json", `{"operation":`, 400, CodeInvalidJSON},
		{"not json", "application/json", `hello`, 400, CodeInvalidJSON},
		{"empty body", "application/json", ``, 400, CodeInvalidJSON},
		{"unknown field", "application/json", `{"operation":"add","operands":[1,2],"x":1}`, 400, CodeInvalidJSON},
		{"trailing value", "application/json", `{"operation":"add","operands":[1,2]}{}`, 400, CodeInvalidJSON},
		{"body too large", "application/json", `{"operation":"add","operands":[1,2],"pad":"` + strings.Repeat("a", 2000) + `"}`, 400, CodeInvalidJSON},
		{"missing content type", "", `{"operation":"add","operands":[1,2]}`, 415, CodeUnsupportedMediaType},
		{"wrong content type", "text/plain", `{"operation":"add","operands":[1,2]}`, 415, CodeUnsupportedMediaType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, "/api/v1/calculate", tc.contentType, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body)
			}
			got := readError(t, rec)
			if got.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tc.wantCode)
			}
			if got.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestOperationsEndpoint(t *testing.T) {
	rec := do(t, newTestRouter(Options{}), http.MethodGet, "/api/v1/operations", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Operations []struct {
			Name   string
			Symbol string
			Arity  int
			Label  string
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Operations) != 7 || got.Operations[5].Name != "sqrt" || got.Operations[5].Arity != 1 {
		t.Errorf("unexpected operations: %+v", got.Operations)
	}
}

func TestHealthz(t *testing.T) {
	rec := do(t, newTestRouter(Options{}), http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestRouter(Options{})
	cases := []struct{ method, path, allow string }{
		{http.MethodGet, "/api/v1/calculate", "POST"},
		{http.MethodDelete, "/api/v1/calculate", "POST"},
		{http.MethodPost, "/api/v1/operations", "GET, HEAD"},
		{http.MethodPost, "/healthz", "GET, HEAD"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, "", "")
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d", rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != tc.allow {
				t.Errorf("Allow = %q, want %q", got, tc.allow)
			}
			if code := readError(t, rec).Code; code != CodeMethodNotAllowed {
				t.Errorf("code = %q", code)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	h := newTestRouter(Options{})
	for _, p := range []string{"/api/v1/nope", "/"} {
		rec := do(t, h, http.MethodGet, p, "", "")
		if rec.Code != http.StatusNotFound || readError(t, rec).Code != CodeNotFound {
			t.Errorf("%s: status = %d body = %s", p, rec.Code, rec.Body)
		}
	}
}

func TestToAPIErrorUnknownIsInternal(t *testing.T) {
	got := toAPIError(http.ErrBodyNotAllowed)
	if got.Status != http.StatusInternalServerError || got.Code != CodeInternal {
		t.Fatalf("got %+v", got)
	}
	if strings.Contains(got.Message, "body") {
		t.Error("internal error message leaks detail")
	}
	if got.Error() != "INTERNAL: internal server error" {
		t.Errorf("Error() = %q", got.Error())
	}
}
