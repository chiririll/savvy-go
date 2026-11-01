package recurring

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"savvy-go/internal/platform/calendar"
)

const dateLayout = "2006-01-02"

// Schedule is when a rule is due: the JSON stored in recurring_transactions.
// No SQL reads it; the database only checks that it is JSON. A new kind of
// schedule is a new key here, not a migration.
type Schedule struct {
	Frequency  string  `json:"frequency"`              // daily, weekly, monthly or yearly
	Interval   int     `json:"interval"`               // every n days, weeks, months or years
	DayOfWeek  *int    `json:"day_of_week,omitempty"`  // 0 Sunday to 6; weekly only
	DayOfMonth *int    `json:"day_of_month,omitempty"` // 1 to 31; monthly only
	Start      string  `json:"start"`                  // the first due date
	End        *string `json:"end,omitempty"`          // the last day a due date may fall on
}

// ErrSchedule is wrapped by every error ParseSchedule and Validate return.
var ErrSchedule = errors.New("invalid schedule")

// ParseSchedule reads and validates the stored JSON.
func ParseSchedule(raw string) (Schedule, error) {
	var s Schedule
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Schedule{}, fmt.Errorf("%w: %v", ErrSchedule, err)
	}
	if err := s.Validate(); err != nil {
		return Schedule{}, err
	}
	return s, nil
}

// JSON is the form stored in the database.
func (s Schedule) JSON() string {
	b, _ := json.Marshal(s)
	return string(b)
}

func date(v string) (time.Time, bool) {
	t, err := time.ParseInLocation(dateLayout, v, time.UTC)
	return t, err == nil && t.Format(dateLayout) == v
}

// Validate checks every field; a schedule that passes can always give a next
// due date.
func (s Schedule) Validate() error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrSchedule}, a...)...)
	}
	switch s.Frequency {
	case "daily", "weekly", "monthly", "yearly":
	default:
		return bad("frequency %q is not daily, weekly, monthly or yearly", s.Frequency)
	}
	if s.Interval < 1 {
		return bad("interval must be at least 1")
	}
	if s.DayOfWeek != nil && (s.Frequency != "weekly" || *s.DayOfWeek < 0 || *s.DayOfWeek > 6) {
		return bad("day_of_week is 0 to 6 and only for a weekly schedule")
	}
	if s.DayOfMonth != nil && (s.Frequency != "monthly" || *s.DayOfMonth < 1 || *s.DayOfMonth > 31) {
		return bad("day_of_month is 1 to 31 and only for a monthly schedule")
	}
	start, ok := date(s.Start)
	if !ok {
		return bad("start %q is not a date (YYYY-MM-DD)", s.Start)
	}
	if s.End != nil {
		end, ok := date(*s.End)
		if !ok {
			return bad("end %q is not a date (YYYY-MM-DD)", *s.End)
		}
		if end.Before(start) {
			return bad("end is before start")
		}
	}
	return nil
}

// Within reports whether a due date is not past the end of the schedule.
func (s Schedule) Within(due string) bool { return s.End == nil || due <= *s.End }

// After is the due date one step after due. Validate must have passed.
func (s Schedule) After(due string) string {
	cur, ok := date(due)
	if !ok {
		return due
	}
	var next time.Time
	switch s.Frequency {
	case "weekly":
		next = cur.AddDate(0, 0, 7*s.Interval)
		if s.DayOfWeek != nil && int(next.Weekday()) != *s.DayOfWeek {
			next = nextWeekday(next, time.Weekday(*s.DayOfWeek))
		}
	case "monthly":
		next = calendar.AddMonths(cur, s.Interval)
		if s.DayOfMonth != nil {
			day := min(*s.DayOfMonth, calendar.DaysInMonth(next))
			next = time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, time.UTC)
		}
	case "yearly":
		next = cur.AddDate(s.Interval, 0, 0)
	default:
		next = cur.AddDate(0, 0, s.Interval)
	}
	return next.Format(dateLayout)
}

func nextWeekday(t time.Time, want time.Weekday) time.Time {
	delta := (int(want) - int(t.Weekday()) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	return t.AddDate(0, 0, delta)
}

// Next is the next due date of a rule whose latest handled due date is last
// (nil when it handled none): the start, or one step after last.
func (s Schedule) Next(last *string) string {
	if last == nil {
		return s.Start
	}
	return s.After(*last)
}
