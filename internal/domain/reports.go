package domain

import (
	"database/sql"
	"strconv"
	"time"
)

type ReportFilter struct {
	PeriodType  string
	PeriodValue string
	StartDate   string
	EndDate     string
	CompareWith string
	AccountIDs  []int64
	CategoryIDs []int64
	TagIDs      []int64
}

type dateRange struct {
	Start time.Time
	End   time.Time
}

type Reports struct {
	DB  *sql.DB
	Loc *time.Location
}

func (s Reports) loc() *time.Location {
	if s.Loc != nil {
		return s.Loc
	}
	return time.UTC
}

func (s Reports) now() time.Time { return time.Now().In(s.loc()) }

func (f ReportFilter) Range(now time.Time) dateRange {
	switch f.PeriodType {
	case "month":
		t := now
		if y, m, ok := parseYearMonth(f.PeriodValue); ok {
			t = time.Date(y, time.Month(m), 1, 0, 0, 0, 0, now.Location())
		}
		return dateRange{startOfMonth(t), endOfMonth(t)}
	case "quarter":
		t := now
		if y, q, ok := parseYearQuarter(f.PeriodValue); ok {
			t = time.Date(y, time.Month((q-1)*3+1), 1, 0, 0, 0, 0, now.Location())
		}
		m := ((int(t.Month())-1)/3)*3 + 1
		start := time.Date(t.Year(), time.Month(m), 1, 0, 0, 0, 0, t.Location())
		return dateRange{start, endOfMonth(start.AddDate(0, 2, 0))}
	case "year":
		y := now.Year()
		if n, err := strconv.Atoi(f.PeriodValue); err == nil && n > 0 {
			y = n
		}
		return dateRange{time.Date(y, 1, 1, 0, 0, 0, 0, now.Location()), time.Date(y, 12, 31, 0, 0, 0, 0, now.Location())}
	case "ytd":
		return dateRange{time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), dateOnly(now)}
	case "custom":
		start, errStart := time.ParseInLocation("2006-01-02", f.StartDate, now.Location())
		end, errEnd := time.ParseInLocation("2006-01-02", f.EndDate, now.Location())
		if errStart != nil || errEnd != nil {
			return defaultRange(now)
		}
		return dateRange{start, end}
	default:
		return defaultRange(now)
	}
}

// defaultRange is the period used whenever none (or an invalid one) is given:
// the last 30 days including today.
func defaultRange(now time.Time) dateRange {
	end := dateOnly(now)
	return dateRange{end.AddDate(0, 0, -29), end}
}

func (f ReportFilter) Comparison(now time.Time) *dateRange {
	if f.CompareWith == "" || f.CompareWith == "none" {
		return nil
	}
	cur := f.Range(now)
	if f.CompareWith == "same_period_last_year" {
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	}
	if f.CompareWith != "previous_period" {
		return nil
	}
	switch f.PeriodType {
	case "month":
		a := addMonthsNoOverflow(cur.Start, -1)
		return &dateRange{startOfMonth(a), endOfMonth(a)}
	case "quarter":
		a := addMonthsNoOverflow(cur.Start, -3)
		m := ((int(a.Month())-1)/3)*3 + 1
		start := time.Date(a.Year(), time.Month(m), 1, 0, 0, 0, 0, a.Location())
		return &dateRange{start, endOfMonth(start.AddDate(0, 2, 0))}
	case "year":
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	case "ytd":
		return &dateRange{cur.Start.AddDate(-1, 0, 0), cur.End.AddDate(-1, 0, 0)}
	default:
		days := int(cur.End.Sub(cur.Start).Hours()/24) + 1
		prevEnd := cur.Start.AddDate(0, 0, -1)
		return &dateRange{prevEnd.AddDate(0, 0, -(days - 1)), prevEnd}
	}
}

func (f ReportFilter) Sparkline(now time.Time, count int) []dateRange {
	cur := f.Range(now)
	days := int(cur.End.Sub(cur.Start).Hours()/24) + 1
	out := make([]dateRange, 0, count)
	for i := count - 1; i >= 0; i-- {
		switch f.PeriodType {
		case "quarter":
			a := addMonthsNoOverflow(cur.Start, -3*i)
			m := ((int(a.Month())-1)/3)*3 + 1
			start := time.Date(a.Year(), time.Month(m), 1, 0, 0, 0, 0, a.Location())
			out = append(out, dateRange{start, endOfMonth(start.AddDate(0, 2, 0))})
		case "year":
			a := cur.Start.AddDate(-i, 0, 0)
			out = append(out, dateRange{time.Date(a.Year(), 1, 1, 0, 0, 0, 0, a.Location()), time.Date(a.Year(), 12, 31, 0, 0, 0, 0, a.Location())})
		case "last_30_days", "custom":
			out = append(out, dateRange{cur.Start.AddDate(0, 0, -days*i), cur.End.AddDate(0, 0, -days*i)})
		case "ytd":
			a := addMonthsNoOverflow(cur.End, -i)
			out = append(out, dateRange{startOfMonth(a), endOfMonth(a)})
		default:
			a := addMonthsNoOverflow(cur.Start, -i)
			out = append(out, dateRange{startOfMonth(a), endOfMonth(a)})
		}
	}
	return out
}

type periodPoint struct{ Key string }

func generatePeriods(start, end time.Time, groupBy string) []periodPoint {
	var out []periodPoint
	cur := start
	for !cur.After(end) {
		switch groupBy {
		case "week":
			wk := startOfWeekSunday(cur)
			out = append(out, periodPoint{Key: wk.Format("2006-01-02")})
			cur = cur.AddDate(0, 0, 7)
		case "month":
			m := startOfMonth(cur)
			out = append(out, periodPoint{Key: m.Format("2006-01-02")})
			cur = addMonthsNoOverflow(cur, 1)
		default:
			out = append(out, periodPoint{Key: cur.Format("2006-01-02")})
			cur = cur.AddDate(0, 0, 1)
		}
	}
	return out
}

func nextPeriod(cur time.Time, groupBy string) time.Time {
	switch groupBy {
	case "week":
		return startOfWeek(cur.AddDate(0, 0, 7))
	case "month":
		return startOfMonth(addMonthsNoOverflow(cur, 1))
	default:
		return cur.AddDate(0, 0, 1)
	}
}

func periodEnd(cur, max time.Time, groupBy string) time.Time {
	var end time.Time
	switch groupBy {
	case "week":
		end = endOfWeek(cur)
	case "month":
		end = endOfMonth(cur)
	default:
		end = cur
	}
	if end.After(max) {
		return max
	}
	return end
}

func startOfWeekSunday(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()-int(t.Weekday()), 0, 0, 0, 0, t.Location())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseYearMonth(s string) (int, int, bool) {
	if len(s) != 7 || s[4] != '-' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	m, err2 := strconv.Atoi(s[5:])
	return y, m, err1 == nil && err2 == nil && m >= 1 && m <= 12
}

func parseYearQuarter(s string) (int, int, bool) {
	if len(s) < 7 || s[4] != '-' || s[5] != 'Q' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	q, err2 := strconv.Atoi(s[6:])
	return y, q, err1 == nil && err2 == nil && q >= 1 && q <= 4
}

func coalesce(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
