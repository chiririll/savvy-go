//go:build audit

package archtest

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestAuditReport lists code above the soft limits, worst first. It never
// fails: it is a worklist for a cleanup session.
//
//	go test -tags audit ./internal/archtest -run Audit -v
func TestAuditReport(t *testing.T) {
	byRule := map[string][]finding{}
	for _, f := range scanRepo(t) {
		byRule[f.rule] = append(byRule[f.rule], f)
	}

	var report strings.Builder
	for _, r := range rules {
		found := byRule[r.name]
		if len(found) == 0 {
			continue
		}
		sort.SliceStable(found, func(i, j int) bool { return found[i].value > found[j].value })
		hard := 0
		for _, f := range found {
			if f.hard() {
				hard++
			}
		}
		fmt.Fprintf(&report, "\n## %s: %s (soft %d, hard %d): %d found, %d over hard\n",
			r.name, r.what, r.soft, r.hard, len(found), hard)
		for _, f := range found {
			fmt.Fprintf(&report, "%5d  %s:%d  %s\n", f.value, f.file, f.line, f.name)
		}
	}
	if report.Len() == 0 {
		report.WriteString("\nNothing above the soft limits.\n")
	}
	t.Log(report.String())
}
