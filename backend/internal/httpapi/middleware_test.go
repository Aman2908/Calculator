package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRecovererReturnsJSON500(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := readError(t, rec); got.Code != CodeInternal || strings.Contains(got.Message, "boom") {
		t.Errorf("unexpected body: %+v", got)
	}
	if !strings.Contains(logs.String(), "boom") {
		t.Error("panic value was not logged")
	}
}

func TestRecovererRepanicsOnAbort(t *testing.T) {
	h := recoverer(slog.Default())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Fatal("expected ErrAbortHandler to propagate")
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestAccessLogRecordsStatus(t *testing.T) {
	var logs bytes.Buffer
	h := NewRouter(nil, Options{Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	out := logs.String()
	for _, want := range []string{"path=/healthz", "status=200", "request_id="} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q missing %q", out, want)
		}
	}
}

func TestCORS(t *testing.T) {
	const origin = "http://localhost:5173"
	h := newTestRouter(Options{CORSOrigin: origin})

	t.Run("preflight from allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/calculate", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Error("missing allow-origin")
		}
	})

	t.Run("other origin gets no CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("unexpected allow-origin")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d", rec.Code)
		}
	})
}

func TestSPAHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":       {Data: []byte("<html>app</html>")},
		"assets/app-1.js":  {Data: []byte("console.log(1)")},
		"assets/style.css": {Data: []byte("body{}")},
	}
	h := spaHandler(fsys)

	cases := []struct {
		method, path, wantBody, wantCache string
		wantStatus                        int
	}{
		{http.MethodGet, "/", "app", "no-cache", 200},
		{http.MethodGet, "/history", "app", "no-cache", 200},
		{http.MethodGet, "/assets/app-1.js", "console.log", "immutable", 200},
		{http.MethodGet, "/assets/", "app", "no-cache", 200},
		{http.MethodGet, "/../../etc/passwd", "invalid URL path", "", 400},
		{http.MethodPost, "/", "METHOD_NOT_ALLOWED", "", 405},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body, tc.wantBody)
			}
			if !strings.Contains(rec.Header().Get("Cache-Control"), tc.wantCache) {
				t.Errorf("Cache-Control = %q, want %q", rec.Header().Get("Cache-Control"), tc.wantCache)
			}
		})
	}
}

func TestRouterServesStaticDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newTestRouter(Options{StaticDir: dir})
	rec := do(t, h, http.MethodGet, "/some/client/route", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "spa") {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	// API routes still win over the SPA fallback.
	rec = do(t, h, http.MethodGet, "/api/v1/unknown", "", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("api 404 status = %d", rec.Code)
	}
}
