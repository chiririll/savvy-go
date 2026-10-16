package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"savvy-go/internal/app"
)

// shutdownTimeout bounds how long Serve waits for in-flight requests after ctx is cancelled.
const shutdownTimeout = 15 * time.Second

// Serve answers requests on the configured address until ctx is cancelled, then drains them.
// The caller closes svc afterwards: requests stop before the queue does, as a request still
// running could enqueue work.
func Serve(ctx context.Context, svc *app.Services) error {
	cfg := svc.Cfg
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           FromServices(svc).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	failed := make(chan error, 1)
	go func() {
		slog.Info("go savvy listening", "addr", cfg.ListenAddr, "data", cfg.DataDir, "version", svc.Build.Version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
