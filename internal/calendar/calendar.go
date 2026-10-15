// Package calendar holds date arithmetic on calendar days.
package calendar

import "time"

// AddMonths moves t by months, keeping the day of the month where the target
// month has it and clamping to the month's last day where it does not
// (31 Jan + 1 month is the last day of February). The result is midnight UTC.
func AddMonths(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := DaysInMonth(first)
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, time.UTC)
}

// DaysInMonth is the number of days in the month t falls in.
func DaysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// StartOfMonth is the first day of t's month, at midnight in t's location.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth is the last day of t's month, at midnight in t's location.
func EndOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location())
}

// StartOfWeek is the Monday on or before t, at midnight in t's location.
func StartOfWeek(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
}

// EndOfWeek is the Sunday on or after t, at midnight in t's location.
func EndOfWeek(t time.Time) time.Time {
	return StartOfWeek(t).AddDate(0, 0, 6)
}
