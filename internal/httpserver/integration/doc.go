// Package integration holds the tests that start a whole app, one per test, and talk to it over
// HTTP. They take about a minute, so they carry the build tag integration:
//
//	go test -tags integration ./internal/httpserver/integration
package integration
