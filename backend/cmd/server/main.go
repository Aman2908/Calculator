// Command server runs the calculator HTTP service.
//
// Configuration (environment):
//
//	PORT         listen port (default 8080)
//	STATIC_DIR   directory with the built SPA to serve at "/" (optional)
//	CORS_ORIGIN  single origin allowed to call the API cross-origin (optional)
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"calculator/internal/calculator"
	"calculator/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

type config struct {
	Port       string
	StaticDir  string
	CORSOrigin string
}

func loadConfig(getenv func(string) string) config {
	cfg := config{
		Port:       getenv("PORT"),
		StaticDir:  getenv("STATIC_DIR"),
		CORSOrigin: getenv("CORS_ORIGIN"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg
}

func newServer(cfg config, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(calculator.Default(), httpapi.Options{
			StaticDir:  cfg.StaticDir,
			CORSOrigin: cfg.CORSOrigin,
			Logger:     logger,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

// run serves on ln until ctx is cancelled, then drains in-flight requests.
func run(ctx context.Context, srv *http.Server, ln net.Listener, logger *slog.Logger) error {
	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", ln.Addr().String())
		errc <- srv.Serve(ln)
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := loadConfig(os.Getenv)
	srv := newServer(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		logger.Error("listen failed", "err", err)
		os.Exit(1)
	}
	if err := run(ctx, srv, ln, logger); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
