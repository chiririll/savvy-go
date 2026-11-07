package seed

import (
	"log/slog"
	"time"
)

// timed runs fn and logs how long it took, so a long seed shows where it is.
func timed(msg string, fn func() error, attrs ...any) error {
	started := time.Now()
	slog.Info(msg+" started", attrs...)
	err := fn()
	attrs = append(attrs, "took", time.Since(started).Round(time.Millisecond))
	if err != nil {
		slog.Error(msg+" failed", append(attrs, "error", err)...)
		return err
	}
	slog.Info(msg+" done", attrs...)
	return nil
}
