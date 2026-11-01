package recurring

import (
	"errors"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestScheduleAfter(t *testing.T) {
	tests := []struct {
		name string
		s    Schedule
		from string
		want string
	}{
		{"daily", Schedule{Frequency: "daily", Interval: 1}, "2024-02-28", "2024-02-29"},
		{"daily crosses a year end", Schedule{Frequency: "daily", Interval: 3}, "2023-12-30", "2024-01-02"},
		{"weekly keeps the weekday", Schedule{Frequency: "weekly", Interval: 2}, "2024-01-01", "2024-01-15"},
		{
			"weekly moves to the chosen weekday",
			Schedule{Frequency: "weekly", Interval: 1, DayOfWeek: ptr(5)}, "2024-01-01", "2024-01-12",
		},
		{
			"weekly on the same weekday",
			Schedule{Frequency: "weekly", Interval: 1, DayOfWeek: ptr(1)}, "2024-01-01", "2024-01-08",
		},
		{"monthly", Schedule{Frequency: "monthly", Interval: 1}, "2024-01-15", "2024-02-15"},
		{
			"monthly clamps to a short month",
			Schedule{Frequency: "monthly", Interval: 1, DayOfMonth: ptr(31)}, "2024-01-31", "2024-02-29",
		},
		{
			"monthly returns to the day after a short month",
			Schedule{Frequency: "monthly", Interval: 1, DayOfMonth: ptr(31)}, "2024-02-29", "2024-03-31",
		},
		{"monthly interval over a year end", Schedule{Frequency: "monthly", Interval: 3}, "2024-11-10", "2025-02-10"},
		{"yearly", Schedule{Frequency: "yearly", Interval: 1}, "2023-06-01", "2024-06-01"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.After(tc.from); got != tc.want {
				t.Errorf("after %s = %s, want %s", tc.from, got, tc.want)
			}
		})
	}
}

func TestNextIsTheStartUntilSomethingWasHandled(t *testing.T) {
	s := Schedule{Frequency: "monthly", Interval: 1, Start: "2026-01-15"}
	if got := s.Next(nil); got != "2026-01-15" {
		t.Errorf("next with nothing handled = %s, want the start", got)
	}
	if got := s.Next(ptr("2026-03-15")); got != "2026-04-15" {
		t.Errorf("next after 2026-03-15 = %s", got)
	}
}

func TestWithin(t *testing.T) {
	open := Schedule{}
	ended := Schedule{End: ptr("2024-03-01")}
	for _, tc := range []struct {
		name string
		s    Schedule
		due  string
		want bool
	}{
		{"no end", open, "2030-01-01", true},
		{"on the end date", ended, "2024-03-01", true},
		{"past the end date", ended, "2024-03-02", false},
	} {
		if got := tc.s.Within(tc.due); got != tc.want {
			t.Errorf("%s: within = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseScheduleRefusesWhatCannotGiveADueDate(t *testing.T) {
	good := `{"frequency":"weekly","interval":2,"day_of_week":3,"start":"2026-01-07","end":"2026-12-31"}`
	s, err := ParseSchedule(good)
	if err != nil || s.Frequency != "weekly" || s.Interval != 2 || *s.DayOfWeek != 3 || *s.End != "2026-12-31" {
		t.Fatalf("schedule = %+v, %v", s, err)
	}
	if again, err := ParseSchedule(s.JSON()); err != nil || again.Start != s.Start {
		t.Fatalf("round trip = %+v, %v", again, err)
	}

	for name, raw := range map[string]string{
		"not json":               `soon`,
		"unknown key":            `{"frequency":"daily","interval":1,"start":"2026-01-01","nth":2}`,
		"unknown frequency":      `{"frequency":"fortnightly","interval":1,"start":"2026-01-01"}`,
		"interval below one":     `{"frequency":"daily","interval":0,"start":"2026-01-01"}`,
		"weekday on a monthly":   `{"frequency":"monthly","interval":1,"day_of_week":2,"start":"2026-01-01"}`,
		"weekday out of range":   `{"frequency":"weekly","interval":1,"day_of_week":7,"start":"2026-01-01"}`,
		"month day out of range": `{"frequency":"monthly","interval":1,"day_of_month":32,"start":"2026-01-01"}`,
		"month day on a weekly":  `{"frequency":"weekly","interval":1,"day_of_month":5,"start":"2026-01-01"}`,
		"start missing":          `{"frequency":"daily","interval":1}`,
		"start is a timestamp":   `{"frequency":"daily","interval":1,"start":"2026-01-01T10:00:00Z"}`,
		"start is not a day":     `{"frequency":"daily","interval":1,"start":"2026-02-30"}`,
		"end is not a date":      `{"frequency":"daily","interval":1,"start":"2026-01-01","end":"never"}`,
		"end before start":       `{"frequency":"daily","interval":1,"start":"2026-01-02","end":"2026-01-01"}`,
	} {
		if _, err := ParseSchedule(raw); !errors.Is(err, ErrSchedule) {
			t.Errorf("%s: err = %v, want ErrSchedule", name, err)
		}
	}
}
