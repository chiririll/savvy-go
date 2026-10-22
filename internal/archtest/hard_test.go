package archtest

import "testing"

// TestCodeStaysWithinHardLimits fails on code that is clearly out of hand.
func TestCodeStaysWithinHardLimits(t *testing.T) {
	for _, f := range scanRepo(t) {
		if !f.hard() {
			continue
		}
		r := rulesByName[f.rule]
		t.Errorf("%s:%d %s: %d %s (hard limit %d). Split it up, or opt out with //archtest:ignore %s <reason>",
			f.file, f.line, f.name, f.value, r.what, r.hard, f.rule)
	}
}
