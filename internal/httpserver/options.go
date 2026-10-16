package httpserver

import "savvy-go/internal/app"

// Option configures New; see the app package.
type Option = app.Option

var (
	WithRates     = app.WithRates
	WithBootstrap = app.WithBootstrap
	WithMailer    = app.WithMailer
	WithQueue     = app.WithQueue
)
