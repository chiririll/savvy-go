package calendar

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestAddMonthsClampsToLastDay(t *testing.T) {
	for _, c := range []struct {
		from   time.Time
		months int
		want   time.Time
	}{
		{day(2024, 1, 31), 1, day(2024, 2, 29)}, // leap year
		{day(2025, 1, 31), 1, day(2025, 2, 28)},
		{day(2025, 3, 31), -1, day(2025, 2, 28)},
		{day(2025, 1, 15), 1, day(2025, 2, 15)},
		{day(2025, 11, 30), 3, day(2026, 2, 28)}, // over a year end
		{day(2025, 5, 10), 0, day(2025, 5, 10)},
	} {
		if got := AddMonths(c.from, c.months); !got.Equal(c.want) {
			t.Errorf(
				"AddMonths(%s, %d) = %s, want %s",
				c.from.Format("2006-01-02"),
				c.months,
				got.Format("2006-01-02"),
				c.want.Format("2006-01-02"),
			)
		}
	}
}

func TestDaysInMonth(t *testing.T) {
	for _, c := range []struct {
		t    time.Time
		want int
	}{
		{day(2024, 2, 10), 29}, {day(2025, 2, 10), 28}, {day(2025, 4, 1), 30}, {day(2025, 12, 31), 31},
	} {
		if got := DaysInMonth(c.t); got != c.want {
			t.Errorf("DaysInMonth(%s) = %d", c.t.Format("2006-01"), got)
		}
	}
}

func TestMonthWeekBounds(t *testing.T) {
	wed := time.Date(2025, 2, 12, 15, 30, 0, 0, time.UTC) // a Wednesday
	for name, c := range map[string]struct{ got, want time.Time }{
		"start of month":             {StartOfMonth(wed), day(2025, 2, 1)},
		"end of month":               {EndOfMonth(wed), day(2025, 2, 28)},
		"start of week":              {StartOfWeek(wed), day(2025, 2, 10)}, // Monday
		"end of week":                {EndOfWeek(wed), day(2025, 2, 16)},   // Sunday
		"monday starts its own week": {StartOfWeek(day(2025, 2, 10)), day(2025, 2, 10)},
		"sunday ends its own week":   {EndOfWeek(day(2025, 2, 16)), day(2025, 2, 16)},
		"week across a month":        {StartOfWeek(day(2025, 3, 2)), day(2025, 2, 24)},
	} {
		if !c.got.Equal(c.want) {
			t.Errorf("%s: got %s want %s", name, c.got.Format("2006-01-02"), c.want.Format("2006-01-02"))
		}
	}
}
