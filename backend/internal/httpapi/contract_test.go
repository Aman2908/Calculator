package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestResponsesMatchOpenAPI validates real handler responses against
// api/openapi.yaml, so the spec and the server cannot drift apart.
func TestResponsesMatchOpenAPI(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("spec is invalid: %v", err)
	}

	h := newTestRouter(Options{})
	cases := []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{"calculate ok", http.MethodPost, "/api/v1/calculate", `{"operation":"divide","operands":[10,4]}`, 200},
		{"calculate unary", http.MethodPost, "/api/v1/calculate", `{"operation":"sqrt","operands":[16]}`, 200},
		{"calculate 400", http.MethodPost, "/api/v1/calculate", `{"operation":"nope","operands":[1,2]}`, 400},
		{"calculate 422", http.MethodPost, "/api/v1/calculate", `{"operation":"divide","operands":[1,0]}`, 422},
		{"operations", http.MethodGet, "/api/v1/operations", "", 200},
		{"health", http.MethodGet, "/healthz", "", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			item := doc.Paths.Find(tc.path)
			if item == nil {
				t.Fatalf("path %s not in spec", tc.path)
			}
			op := item.GetOperation(tc.method)
			if op == nil {
				t.Fatalf("%s %s not in spec", tc.method, tc.path)
			}
			resp := op.Responses.Status(rec.Code)
			if resp == nil {
				t.Fatalf("status %d not documented for %s %s", rec.Code, tc.method, tc.path)
			}
			schema := resp.Value.Content.Get("application/json").Schema.Value

			var body any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if err := schema.VisitJSON(body); err != nil {
				t.Errorf("response does not match spec: %v\nbody: %s", err, rec.Body)
			}
		})
	}
}
