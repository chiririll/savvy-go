package domain

import (
	"context"
	"database/sql"
	"math"
	"time"
)

type BudgetProgress struct {
	Spent       float64
	Remaining   float64
	Percent     float64
	PeriodStart string
	PeriodEnd   string
	IsExceeded  bool
}

func (p BudgetProgress) JSON() map[string]any {
	return map[string]any{
		"spent": p.Spent, "remaining": p.Remaining, "percent": p.Percent,
		"period_start": p.PeriodStart, "period_end": p.PeriodEnd,
		"is_exceeded": p.IsExceeded,
	}
}

type Budget struct {
	ID              int64
	Name            string
	Amount          float64
	CurrencyID      *int64
	Period          string
	StartDate       *string
	EndDate         *string
	IsGlobal        bool
	NotifyAtPercent *int
	IsActive        bool
	Currency        *Currency
	Categories      []Category
	Tags            []Tag
	Progress        *BudgetProgress
}

func periodLabel(p string) string {
	switch p {
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	case "yearly":
		return "Yearly"
	case "one_time":
		return "One-time"
	default:
		return p
	}
}

func (b Budget) JSON() map[string]any {
	m := map[string]any{
		"id": b.ID, "name": b.Name, "amount": b.Amount,
		"currencyId": b.CurrencyID, "period": b.Period, "periodLabel": periodLabel(b.Period),
		"startDate": b.StartDate, "endDate": b.EndDate,
		"isGlobal": b.IsGlobal, "notifyAtPercent": b.NotifyAtPercent,
		"isActive": b.IsActive,
		"categories": mapSliceVal(b.Categories, Category.JSON),
		"tags":       mapSliceVal(b.Tags, Tag.JSON),
	}
	if b.Currency != nil {
		m["currency"] = b.Currency.JSON()
	}
	if b.Progress != nil {
		m["progress"] = b.Progress.JSON()
	}
	return m
}

type BudgetInput struct {
	Name            string
	Amount          float64
	CurrencyID      *int64
	Period          string
	StartDate       *string
	EndDate         *string
	IsGlobal        *bool
	NotifyAtPercent *int
	IsActive        *bool
	CategoryIDs     []int64
	TagIDs          []int64
	HasCategoryIDs  bool
	HasTagIDs       bool
}

type Budgets struct{ DB *sql.DB }

func (s Budgets) All(ctx context.Context) ([]Budget, error) {
	return s.list(ctx, ``)
}

func (s Budgets) ByID(ctx context.Context, id int64) (*Budget, error) {
	list, err := s.list(ctx, `WHERE b.id = ?`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Budgets) Create(ctx context.Context, in BudgetInput) (*Budget, error) {
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	global := false
	if in.IsGlobal != nil {
		global = *in.IsGlobal
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO budgets (name, amount, currency_id, period, start_date, end_date,
			is_global, notify_at_percent, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		in.Name, in.Amount, in.CurrencyID, in.Period, in.StartDate, in.EndDate,
		boolInt(global), in.NotifyAtPercent, boolInt(active), now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := s.saveCats(ctx, id, in.CategoryIDs); err != nil {
		return nil, err
	}
	if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Budgets) Update(ctx context.Context, id int64, in BudgetInput) (*Budget, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if in.Name == "" {
		in.Name = cur.Name
	}
	if in.Amount == 0 {
		in.Amount = cur.Amount
	}
	if in.Period == "" {
		in.Period = cur.Period
	}
	if in.CurrencyID == nil {
		in.CurrencyID = cur.CurrencyID
	}
	if in.StartDate == nil {
		in.StartDate = cur.StartDate
	}
	if in.EndDate == nil {
		in.EndDate = cur.EndDate
	}
	global := cur.IsGlobal
	if in.IsGlobal != nil {
		global = *in.IsGlobal
	}
	active := cur.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	notify := cur.NotifyAtPercent
	if in.NotifyAtPercent != nil {
		notify = in.NotifyAtPercent
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE budgets SET name=?, amount=?, currency_id=?, period=?, start_date=?, end_date=?,
			is_global=?, notify_at_percent=?, is_active=?, updated_at=? WHERE id=?`,
		in.Name, in.Amount, in.CurrencyID, in.Period, in.StartDate, in.EndDate,
		boolInt(global), notify, boolInt(active), now, id)
	if err != nil {
		return nil, err
	}
	if in.HasCategoryIDs {
		if err := s.saveCats(ctx, id, in.CategoryIDs); err != nil {
			return nil, err
		}
	}
	if in.HasTagIDs {
		if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s Budgets) Delete(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM budgets WHERE id = ?`, id)
	return err
}

func (s Budgets) CalculateProgress(ctx context.Context, b *Budget) BudgetProgress {
	start, end := budgetPeriodDates(*b)
	spent := s.spent(ctx, b, start, end)
	remaining := math.Max(0, b.Amount-spent)
	percent := 0.0
	if b.Amount > 0 {
		percent = math.Round((spent/b.Amount)*1000) / 10
	}
	return BudgetProgress{
		Spent: spent, Remaining: remaining, Percent: percent,
		PeriodStart: start.Format("2006-01-02"), PeriodEnd: end.Format("2006-01-02"),
		IsExceeded: spent > b.Amount,
	}
}

func (s Budgets) spent(ctx context.Context, b *Budget, start, end time.Time) float64 {
	targetRate := 1.0
	if b.Currency != nil && b.Currency.Rate > 0 {
		targetRate = b.Currency.Rate
	} else if b.CurrencyID == nil {
		if base, _ := (Currencies{DB: s.DB}).Base(ctx); base != nil && base.Rate > 0 {
			targetRate = base.Rate
		}
	}

	q := `
		SELECT COALESCE(SUM(t.amount * c.rate), 0)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		JOIN currencies c ON c.id = a.currency_id
		WHERE t.status = 'confirmed' AND t.type = 'expense'
			AND t.date >= ? AND t.date <= ?`
	args := []any{start.Format("2006-01-02"), end.Format("2006-01-02")}
	if !b.IsGlobal {
		if len(b.Categories) == 0 {
			return 0
		}
		q += ` AND t.category_id IN (`
		for i, c := range b.Categories {
			if i > 0 {
				q += `,`
			}
			q += `?`
			args = append(args, c.ID)
		}
		q += `)`
	}
	if len(b.Tags) > 0 {
		q += ` AND EXISTS (
			SELECT 1 FROM transaction_tag tt
			WHERE tt.transaction_id = t.id AND tt.tag_id IN (`
		for i, t := range b.Tags {
			if i > 0 {
				q += `,`
			}
			q += `?`
			args = append(args, t.ID)
		}
		q += `))`
	}
	var total sql.NullFloat64
	if err := s.DB.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0
	}
	if targetRate > 0 {
		return total.Float64 / targetRate
	}
	return total.Float64
}

func budgetPeriodDates(b Budget) (time.Time, time.Time) {
	now := time.Now()
	anchor := now
	if b.StartDate != nil && *b.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", *b.StartDate, now.Location()); err == nil {
			anchor = t
		}
	}
	switch b.Period {
	case "one_time":
		start := startOfMonth(now)
		end := endOfMonth(now)
		if b.StartDate != nil && *b.StartDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *b.StartDate, now.Location()); err == nil {
				start = t
			}
		}
		if b.EndDate != nil && *b.EndDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *b.EndDate, now.Location()); err == nil {
				end = t
			}
		}
		return start, end
	case "weekly":
		return startOfWeek(anchor), endOfWeek(anchor)
	case "yearly":
		return time.Date(anchor.Year(), 1, 1, 0, 0, 0, 0, anchor.Location()),
			time.Date(anchor.Year(), 12, 31, 0, 0, 0, 0, anchor.Location())
	default: // monthly
		return startOfMonth(anchor), endOfMonth(anchor)
	}
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func endOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	// Carbon default: Monday.
	offset := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
}

func endOfWeek(t time.Time) time.Time {
	return startOfWeek(t).AddDate(0, 0, 6)
}

func (s Budgets) list(ctx context.Context, where string, args ...any) ([]Budget, error) {
	q := `SELECT b.id, b.name, b.amount, b.currency_id, b.period, b.start_date, b.end_date,
		b.is_global, b.notify_at_percent, b.is_active FROM budgets b ` + where + ` ORDER BY b.id`
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	curs := Currencies{DB: s.DB}
	for i := range out {
		if out[i].CurrencyID != nil {
			if c, _ := curs.ByID(ctx, *out[i].CurrencyID); c != nil {
				out[i].Currency = c
			}
		}
		out[i].Categories, _ = s.cats(ctx, out[i].ID)
		out[i].Tags, _ = s.tags(ctx, out[i].ID)
		p := s.CalculateProgress(ctx, &out[i])
		out[i].Progress = &p
	}
	return out, nil
}

func (s Budgets) cats(ctx context.Context, id int64) ([]Category, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.name, c.type, c.icon, c.color, 0
		FROM categories c JOIN budget_category bc ON bc.category_id = c.id
		WHERE bc.budget_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s Budgets) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT tags.id, tags.name, tags.created_at, 0 FROM tags
		JOIN budget_tag bt ON bt.tag_id = tags.id WHERE bt.budget_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s Budgets) saveCats(ctx context.Context, id int64, ids []int64) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM budget_category WHERE budget_id = ?`, id); err != nil {
		return err
	}
	for _, catID := range ids {
		if _, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO budget_category (budget_id, category_id) VALUES (?,?)`, id, catID); err != nil {
			return err
		}
	}
	return nil
}

func (s Budgets) saveTags(ctx context.Context, id int64, ids []int64) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM budget_tag WHERE budget_id = ?`, id); err != nil {
		return err
	}
	for _, tagID := range ids {
		if _, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO budget_tag (budget_id, tag_id) VALUES (?,?)`, id, tagID); err != nil {
			return err
		}
	}
	return nil
}

func scanBudget(row interface{ Scan(...any) error }) (Budget, error) {
	var b Budget
	var curID, notify sql.NullInt64
	var start, end sql.NullString
	var global, active int
	err := row.Scan(&b.ID, &b.Name, &b.Amount, &curID, &b.Period, &start, &end, &global, &notify, &active)
	if curID.Valid {
		b.CurrencyID = &curID.Int64
	}
	if start.Valid {
		b.StartDate = &start.String
	}
	if end.Valid {
		b.EndDate = &end.String
	}
	if notify.Valid {
		v := int(notify.Int64)
		b.NotifyAtPercent = &v
	}
	b.IsGlobal = global != 0
	b.IsActive = active != 0
	return b, err
}
