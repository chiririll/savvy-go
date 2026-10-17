package reports

import (
	"testing"
	"time"
)

// now is a Wednesday in the second quarter.
var now = time.Date(2024, 5, 15, 13, 30, 0, 0, time.UTC)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func span(r dateRange) string {
	return r.Start.Format("2006-01-02") + ".." + r.End.Format("2006-01-02")
}

func TestRange(t *testing.T) {
	tests := []struct {
		name string
		f    Filter
		want string
	}{
		{"a named month", Filter{PeriodType: "month", PeriodValue: "2024-02"}, "2024-02-01..2024-02-29"},
		{"a month without a value is this month", Filter{PeriodType: "month"}, "2024-05-01..2024-05-31"},
		{"a month out of range is this month", Filter{PeriodType: "month", PeriodValue: "2024-13"}, "2024-05-01..2024-05-31"},
		{"a named quarter", Filter{PeriodType: "quarter", PeriodValue: "2023-Q4"}, "2023-10-01..2023-12-31"},
		{"a quarter without a value is this quarter", Filter{PeriodType: "quarter"}, "2024-04-01..2024-06-30"},
		{"a quarter past Q4 is this quarter", Filter{PeriodType: "quarter", PeriodValue: "2024-Q5"}, "2024-04-01..2024-06-30"},
		{"a named year", Filter{PeriodType: "year", PeriodValue: "2023"}, "2023-01-01..2023-12-31"},
		{"a year without a value is this year", Filter{PeriodType: "year"}, "2024-01-01..2024-12-31"},
		{"year to date ends today", Filter{PeriodType: "ytd"}, "2024-01-01..2024-05-15"},
		{"a custom range", Filter{PeriodType: "custom", StartDate: "2024-01-13", EndDate: "2024-01-15"}, "2024-01-13..2024-01-15"},
		{"a custom range with a bad date is the default", Filter{PeriodType: "custom", StartDate: "2024-01-13", EndDate: "tomorrow"}, "2024-04-16..2024-05-15"},
		{"no period is the last 30 days", Filter{}, "2024-04-16..2024-05-15"},
		{"an unknown period is the last 30 days", Filter{PeriodType: "decade"}, "2024-04-16..2024-05-15"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := span(tc.f.Range(now)); got != tc.want {
				t.Errorf("range = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestComparison(t *testing.T) {
	tests := []struct {
		name string
		f    Filter
		want string // empty: no comparison
	}{
		{"unset", Filter{PeriodType: "month"}, ""},
		{"none", Filter{PeriodType: "month", CompareWith: "none"}, ""},
		{"unknown", Filter{PeriodType: "month", CompareWith: "tomorrow"}, ""},
		{"previous month", Filter{PeriodType: "month", PeriodValue: "2024-03", CompareWith: "previous_period"}, "2024-02-01..2024-02-29"},
		{"previous month over a year end", Filter{PeriodType: "month", PeriodValue: "2024-01", CompareWith: "previous_period"}, "2023-12-01..2023-12-31"},
		{"previous quarter", Filter{PeriodType: "quarter", PeriodValue: "2024-Q1", CompareWith: "previous_period"}, "2023-10-01..2023-12-31"},
		{"previous year", Filter{PeriodType: "year", PeriodValue: "2024", CompareWith: "previous_period"}, "2023-01-01..2023-12-31"},
		{"previous year to date", Filter{PeriodType: "ytd", CompareWith: "previous_period"}, "2023-01-01..2023-05-15"},
		{"previous custom range is as long and adjacent", Filter{PeriodType: "custom", StartDate: "2024-01-11", EndDate: "2024-01-15", CompareWith: "previous_period"}, "2024-01-06..2024-01-10"},
		{"same period last year", Filter{PeriodType: "month", PeriodValue: "2024-03", CompareWith: "same_period_last_year"}, "2023-03-01..2023-03-31"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.Comparison(now)
			switch {
			case tc.want == "" && got != nil:
				t.Errorf("comparison = %s, want none", span(*got))
			case tc.want != "" && got == nil:
				t.Errorf("no comparison, want %s", tc.want)
			case got != nil && span(*got) != tc.want:
				t.Errorf("comparison = %s, want %s", span(*got), tc.want)
			}
		})
	}
}

// The sparkline ends with the period itself and walks back one period at a time.
func TestSparkline(t *testing.T) {
	tests := []struct {
		name string
		f    Filter
		want []string
	}{
		{"months", Filter{PeriodType: "month", PeriodValue: "2024-02"}, []string{"2023-12-01..2023-12-31", "2024-01-01..2024-01-31", "2024-02-01..2024-02-29"}},
		{"quarters", Filter{PeriodType: "quarter", PeriodValue: "2024-Q1"}, []string{"2023-07-01..2023-09-30", "2023-10-01..2023-12-31", "2024-01-01..2024-03-31"}},
		{"years", Filter{PeriodType: "year", PeriodValue: "2024"}, []string{"2022-01-01..2022-12-31", "2023-01-01..2023-12-31", "2024-01-01..2024-12-31"}},
		{"year to date goes by month", Filter{PeriodType: "ytd"}, []string{"2024-03-01..2024-03-31", "2024-04-01..2024-04-30", "2024-05-01..2024-05-31"}},
		{"custom ranges step back by their own length", Filter{PeriodType: "custom", StartDate: "2024-01-11", EndDate: "2024-01-15"}, []string{"2024-01-01..2024-01-05", "2024-01-06..2024-01-10", "2024-01-11..2024-01-15"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.Sparkline(now, 3)
			if len(got) != len(tc.want) {
				t.Fatalf("%d points, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if span(got[i]) != tc.want[i] {
					t.Errorf("point %d = %s, want %s", i, span(got[i]), tc.want[i])
				}
			}
		})
	}
}

func keys(ps []periodPoint) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Key
	}
	return out
}

func TestGeneratePeriods(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		groupBy    string
		want       []string
	}{
		{"days", "2024-02-27", "2024-03-01", "day", []string{"2024-02-27", "2024-02-28", "2024-02-29", "2024-03-01"}},
		{"an unknown grouping is days", "2024-01-01", "2024-01-02", "decade", []string{"2024-01-01", "2024-01-02"}},
		// Weeks start on Sunday here, as in the SQL that groups the totals (see filter.PeriodExpr).
		{"weeks start on Sunday", "2024-01-03", "2024-01-17", "week", []string{"2023-12-31", "2024-01-07", "2024-01-14"}},
		{"a range ending on a Sunday includes that week", "2024-01-03", "2024-01-07", "week", []string{"2023-12-31", "2024-01-07"}},
		{"months start on the first", "2024-01-31", "2024-04-02", "month", []string{"2024-01-01", "2024-02-01", "2024-03-01", "2024-04-01"}},
		{"a single day", "2024-01-05", "2024-01-05", "day", []string{"2024-01-05"}},
		{"an empty range", "2024-01-06", "2024-01-05", "day", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := keys(generatePeriods(day(tc.start), day(tc.end), tc.groupBy))
			if len(got) != len(tc.want) {
				t.Fatalf("periods %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("periods %v, want %v", got, tc.want)
					break
				}
			}
		})
	}
}

func TestNextPeriodAndPeriodEnd(t *testing.T) {
	if got := nextPeriod(day("2024-01-31"), "month").Format("2006-01-02"); got != "2024-02-01" {
		t.Errorf("next month %s", got)
	}
	if got := nextPeriod(day("2024-01-31"), "day").Format("2006-01-02"); got != "2024-02-01" {
		t.Errorf("next day %s", got)
	}
	if got := nextPeriod(day("2024-01-10"), "week").Weekday(); got != time.Monday {
		t.Errorf("next week starts on %s", got)
	}
	max := day("2024-01-20")
	if got := periodEnd(day("2024-01-10"), max, "month"); !got.Equal(max) {
		t.Errorf("a period never runs past the end of the range: %s", got)
	}
	if got := periodEnd(day("2024-01-10"), day("2024-03-01"), "month").Format("2006-01-02"); got != "2024-01-31" {
		t.Errorf("month end %s", got)
	}
	if got := periodEnd(day("2024-01-10"), day("2024-03-01"), "day").Format("2006-01-02"); got != "2024-01-10" {
		t.Errorf("day end %s", got)
	}
}

func TestParsePeriodValues(t *testing.T) {
	for in, want := range map[string]bool{"2024-01": true, "2024-12": true, "2024-00": false, "2024-13": false, "24-01": false, "2024/01": false, "abcd-01": false, "": false} {
		if _, _, ok := parseYearMonth(in); ok != want {
			t.Errorf("parseYearMonth(%q) ok = %v, want %v", in, ok, want)
		}
	}
	for in, want := range map[string]bool{"2024-Q1": true, "2024-Q4": true, "2024-Q0": false, "2024-Q5": false, "2024-q1": false, "2024-1": false, "": false} {
		if _, _, ok := parseYearQuarter(in); ok != want {
			t.Errorf("parseYearQuarter(%q) ok = %v, want %v", in, ok, want)
		}
	}
}
