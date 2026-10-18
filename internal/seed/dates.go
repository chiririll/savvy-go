package seed

import (
	"math"
	"time"
)

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

func dateOn(t time.Time, day int) time.Time {
	return time.Date(t.Year(), t.Month(), day, 0, 0, 0, 0, t.Location())
}

func addMonthsNoOverflow(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, t.Location())
	last := daysInMonth(first)
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, t.Location())
}

func nextMonthOnDay(now time.Time, day int) time.Time {
	d := dateOn(now, minInt(day, daysInMonth(now)))
	return addMonthsNoOverflow(d, 1)
}

func monthsBetween(from, to time.Time) int {
	return (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
}

func round2(v float64) float64 { return roundN(v, 2) }

func roundN(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
