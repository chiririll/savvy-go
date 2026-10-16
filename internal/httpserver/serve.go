package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/config"
	"savvy-go/internal/locator"
)

// shutdownTimeout bounds how long Serve waits for in-flight requests after ctx is cancelled.
const shutdownTimeout = 15 * time.Second

// Serve answers requests on the configured address until ctx is cancelled, then drains them.
// The caller closes the locator afterwards: requests stop before the queue does, as a request still
// running could enqueue work.
func Serve(ctx context.Context, l *locator.Locator) error {
	cfg := locator.Get[appconfig.Config](l)
	listen := config.Of(locator.Get[*config.Config](l), ServerSection).Get().Listen
	srv := &http.Server{
		Addr:              listen,
		Handler:           FromLocator(l).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	failed := make(chan error, 1)
	go func() {
		slog.Info("go savvy listening", "addr", listen, "data", cfg.DataDir, "version", locator.Get[appconfig.Build](l).Version)
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
