// Package httpapi exposes the calculator over HTTP/JSON and, optionally,
// serves the built single-page app from the same origin.
package httpapi

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"calculator/internal/calculator"
)

// Options configures the router. Zero values are valid: no static files,
// no CORS, logs discarded.
type Options struct {
	// StaticDir, when set, is served at "/" with index.html as the fallback
	// for client-side routes.
	StaticDir string
	// CORSOrigin, when set, is the single origin allowed to call the API
	// cross-origin (used by the Vite dev server when not proxying).
	CORSOrigin string
	Logger     *slog.Logger
}

// NewRouter wires routes and middleware.
func NewRouter(reg *calculator.Registry, opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	h := &handlers{reg: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/calculate", h.calculate)
	mux.HandleFunc("GET /api/v1/operations", h.operations)
	mux.HandleFunc("GET /healthz", health)

	// Same paths without a method: reached only when the method is wrong,
	// so we can answer with a JSON 405 instead of the mux's plain text.
	mux.Handle("/api/v1/calculate", methodNotAllowed("POST"))
	mux.Handle("/api/v1/operations", methodNotAllowed("GET, HEAD"))
	mux.Handle("/healthz", methodNotAllowed("GET, HEAD"))
	mux.Handle("/api/", http.HandlerFunc(notFound))

	if opts.StaticDir != "" {
		mux.Handle("/", spaHandler(os.DirFS(opts.StaticDir)))
	} else {
		mux.Handle("/", http.HandlerFunc(notFound))
	}

	mws := []middleware{requestID, accessLog(logger), recoverer(logger)}
	if opts.CORSOrigin != "" {
		mws = append(mws, cors(opts.CORSOrigin))
	}
	return chain(mux, mws...)
}

func methodNotAllowed(allow string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, &APIError{
			Status:  http.StatusMethodNotAllowed,
			Code:    CodeMethodNotAllowed,
			Message: r.Method + " is not allowed; use " + allow,
		})
	})
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, &APIError{Status: http.StatusNotFound, Code: CodeNotFound, Message: "no route for " + r.URL.Path})
}

// spaHandler serves real files from fsys and falls back to index.html for
// any other GET, so client-side routes survive a page reload. Hashed assets
// are cached forever; index.html is always revalidated.
func spaHandler(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed("GET, HEAD").ServeHTTP(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if isFile(fsys, name) {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, fsys, "index.html")
	})
}

// isFile reports whether name is a regular file in fsys. An empty or invalid
// name (e.g. the root "/") is not a file.
func isFile(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}
