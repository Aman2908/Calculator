package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	env := map[string]string{"STATIC_DIR": "/static", "CORS_ORIGIN": "http://localhost:5173"}
	cfg := loadConfig(func(k string) string { return env[k] })
	if cfg.Port != "8080" || cfg.StaticDir != "/static" || cfg.CORSOrigin != "http://localhost:5173" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	env["PORT"] = "9090"
	if got := loadConfig(func(k string) string { return env[k] }).Port; got != "9090" {
		t.Fatalf("port = %q", got)
	}
}

func TestNewServerHasTimeouts(t *testing.T) {
	srv := newServer(config{Port: "1234"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if srv.Addr != ":1234" {
		t.Errorf("addr = %q", srv.Addr)
	}
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Error("server timeouts must be set")
	}
}

func TestRunServesAndShutsDownGracefully(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, newServer(config{}, logger), ln, logger) }()

	resp, err := http.Post("http://"+ln.Addr().String()+"/api/v1/calculate", "application/json",
		strings.NewReader(`{"operation":"add","operands":[2,2]}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"result":4`) {
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestRunReturnsServeError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = ln.Close() // Serve on a closed listener fails immediately.
	if err := run(context.Background(), newServer(config{}, logger), ln, logger); err == nil {
		t.Fatal("expected an error")
	}
}
