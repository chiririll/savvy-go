// Package archtest keeps the Go code readable. It has no production code:
// the checks are tests.
//
// Every metric has a hard limit and a soft limit (see rules in scan_test.go).
//
//   - Hard limits run with every `go test ./...`. They catch code that is
//     clearly out of hand and fail the build.
//
//   - Soft limits run only on request and never fail. They print a report to
//     look for code worth cleaning up:
//
//     go test -tags audit ./internal/archtest -run Audit -v
//
// A declaration can opt out of one rule with a line in its doc comment:
//
//	//archtest:ignore funcLines parser tables read best in one piece
//
// The reason is required.
package archtest
