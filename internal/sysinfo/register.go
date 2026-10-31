package sysinfo

import "savvy-go/internal/locator"

// Register adds the host information service. It needs nothing else.
func Register(l *locator.Locator) {
	locator.Provide(l, func(*locator.Locator) (Service, error) {
		return Service{Collector: Default()}, nil
	})
}
