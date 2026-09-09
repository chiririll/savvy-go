package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Recurring struct {
	ID          int64
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      float64
	ToAmount    *float64
	Description *string
	Frequency   string
	Interval    int
	DayOfWeek   *int
	DayOfMonth  *int
	StartDate   string
	EndDate     *string
	NextRunDate string
	LastRunDate *string
	IsActive    bool
	Account     *Account
	ToAccount   *Account
	Category    *Category
	Tags        []Tag
}

func frequencyLabel(f string) string {
	switch f {
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	case "yearly":
		return "Yearly"
	default:
		return f
	}
}

func (r Recurring) JSON() map[string]any {
	m := map[string]any{
		"id": r.ID, "type": r.Type, "accountId": r.AccountID,
		"amount": r.Amount, "description": r.Description,
		"frequency": r.Frequency, "frequencyLabel": frequencyLabel(r.Frequency),
		"interval": r.Interval, "startDate": r.StartDate,
		"nextRunDate": r.NextRunDate, "isActive": r.IsActive,
		"tags": mapSliceVal(r.Tags, Tag.JSON),
	}
	if r.ToAccountID != nil {
		m["toAccountId"] = *r.ToAccountID
	}
	if r.CategoryID != nil {
		m["categoryId"] = *r.CategoryID
	}
	if r.ToAmount != nil {
		m["toAmount"] = *r.ToAmount
	}
	if r.DayOfWeek != nil {
		m["dayOfWeek"] = *r.DayOfWeek
	}
	if r.DayOfMonth != nil {
		m["dayOfMonth"] = *r.DayOfMonth
	}
	if r.EndDate != nil {
		m["endDate"] = *r.EndDate
	}
	if r.LastRunDate != nil {
		m["lastRunDate"] = *r.LastRunDate
	}
	if r.Account != nil {
		m["account"] = r.Account.JSON()
	}
	if r.ToAccount != nil {
		m["toAccount"] = r.ToAccount.JSON()
	}
	if r.Category != nil {
		m["category"] = r.Category.JSON()
	}
	return m
}

type RecurringInput struct {
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      float64
	ToAmount    *float64
	Description *string
	Frequency   string
	Interval    int
	DayOfWeek   *int
	DayOfMonth  *int
	StartDate   string
	EndDate     *string
	IsActive    *bool
	TagIDs      []int64
	HasTagIDs   bool
}

type RecurringStore struct {
	DB  *sql.DB
	Txs Transactions
}

func (s RecurringStore) All(ctx context.Context) ([]Recurring, error) {
	return s.list(ctx, `ORDER BY next_run_date, id`)
}

func (s RecurringStore) Upcoming(ctx context.Context, limit int) ([]Recurring, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.list(ctx, `WHERE is_active = 1 ORDER BY next_run_date, id LIMIT ?`, limit)
}

func (s RecurringStore) ByID(ctx context.Context, id int64) (*Recurring, error) {
	list, err := s.list(ctx, `WHERE id = ?`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s RecurringStore) Create(ctx context.Context, in RecurringInput) (*Recurring, error) {
	if in.Interval < 1 {
		in.Interval = 1
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO recurring_transactions (
			type, account_id, to_account_id, category_id, amount, to_amount, description,
			frequency, interval, day_of_week, day_of_month, start_date, end_date,
			next_run_date, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.Type, in.AccountID, in.ToAccountID, in.CategoryID, in.Amount, in.ToAmount, in.Description,
		in.Frequency, in.Interval, in.DayOfWeek, in.DayOfMonth, in.StartDate, in.EndDate,
		in.StartDate, boolInt(active), now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	rec, err := s.ByID(ctx, id)
	if err != nil || rec == nil {
		return rec, err
	}
	if rec.withinSchedule() {
		if _, err := s.createPending(ctx, rec); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s RecurringStore) Update(ctx context.Context, id int64, in RecurringInput) (*Recurring, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if in.Type == "" {
		in.Type = cur.Type
	}
	if in.AccountID == 0 {
		in.AccountID = cur.AccountID
	}
	if in.Amount == 0 {
		in.Amount = cur.Amount
	}
	if in.Frequency == "" {
		in.Frequency = cur.Frequency
	}
	if in.Interval < 1 {
		in.Interval = cur.Interval
	}
	if in.StartDate == "" {
		in.StartDate = cur.StartDate
	}
	next := cur.NextRunDate
	if in.StartDate != cur.StartDate {
		next = in.StartDate
	}
	active := cur.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE recurring_transactions SET type=?, account_id=?, to_account_id=?, category_id=?,
			amount=?, to_amount=?, description=?, frequency=?, interval=?, day_of_week=?,
			day_of_month=?, start_date=?, end_date=?, next_run_date=?, is_active=?, updated_at=?
		WHERE id=?`,
		in.Type, in.AccountID, in.ToAccountID, in.CategoryID, in.Amount, in.ToAmount, in.Description,
		in.Frequency, in.Interval, in.DayOfWeek, in.DayOfMonth, in.StartDate, in.EndDate,
		next, boolInt(active), now, id)
	if err != nil {
		return nil, err
	}
	if in.HasTagIDs {
		if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	rec, err := s.ByID(ctx, id)
	if err != nil || rec == nil {
		return rec, err
	}
	if err := s.syncOpenPending(ctx, rec); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s RecurringStore) Delete(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM transactions WHERE recurring_transaction_id = ? AND status = 'pending'`, id)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM recurring_transactions WHERE id = ?`, id)
	return err
}

func (s RecurringStore) AdvanceAfterOccurrence(ctx context.Context, id int64) error {
	rec, err := s.ByID(ctx, id)
	if err != nil || rec == nil {
		return err
	}
	next := rec.calculateNextRunDate()
	now := time.Now().UTC()
	_, err = s.DB.ExecContext(ctx, `
		UPDATE recurring_transactions SET last_run_date=?, next_run_date=?, updated_at=? WHERE id=?`,
		now.Format("2006-01-02"), next, now.Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	rec, err = s.ByID(ctx, id)
	if err != nil || rec == nil {
		return err
	}
	if !rec.withinSchedule() {
		return nil
	}
	pending, err := s.pendingID(ctx, id)
	if err != nil || pending != 0 {
		return err
	}
	_, err = s.createPending(ctx, rec)
	return err
}

// EnsureUpcoming creates a pending occurrence for every active template that
// is still on schedule and has none (scheduler / crash recovery).
func (s RecurringStore) EnsureUpcoming(ctx context.Context) error {
	list, err := s.list(ctx, `WHERE is_active = 1`)
	if err != nil {
		return err
	}
	for i := range list {
		rec := &list[i]
		if !rec.withinSchedule() {
			continue
		}
		pending, err := s.pendingID(ctx, rec.ID)
		if err != nil {
			return err
		}
		if pending != 0 {
			continue
		}
		if _, err := s.createPending(ctx, rec); err != nil {
			return fmt.Errorf("recurring %d: %w", rec.ID, err)
		}
	}
	return nil
}

func (s RecurringStore) createPending(ctx context.Context, rec *Recurring) (*Transaction, error) {
	var toAmt *float64
	if rec.Type == "transfer" {
		if rec.ToAmount != nil {
			v := *rec.ToAmount
			toAmt = &v
		} else {
			v, err := s.calculateToAmount(ctx, rec)
			if err != nil {
				return nil, err
			}
			toAmt = &v
		}
	}
	status := "pending"
	date := rec.NextRunDate
	in := TxInput{
		Type: rec.Type, AccountID: rec.AccountID, ToAccountID: rec.ToAccountID,
		CategoryID: rec.CategoryID, Amount: rec.Amount, ToAmount: toAmt,
		Description: rec.Description, Date: &date, Status: &status, RecurringID: &rec.ID,
	}
	for _, t := range rec.Tags {
		in.TagIDs = append(in.TagIDs, t.ID)
	}
	return s.Txs.Create(ctx, in)
}

func (s RecurringStore) syncOpenPending(ctx context.Context, rec *Recurring) error {
	pendingID, err := s.pendingID(ctx, rec.ID)
	if err != nil {
		return err
	}
	if pendingID == 0 {
		if rec.withinSchedule() {
			_, err = s.createPending(ctx, rec)
			return err
		}
		return nil
	}
	var toAmt any
	if rec.Type == "transfer" {
		if rec.ToAmount != nil {
			toAmt = *rec.ToAmount
		} else {
			v, err := s.calculateToAmount(ctx, rec)
			if err != nil {
				return err
			}
			toAmt = v
		}
	} else {
		toAmt = rec.ToAmount
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.DB.ExecContext(ctx, `
		UPDATE transactions SET type=?, account_id=?, to_account_id=?, category_id=?,
			amount=?, to_amount=?, description=?, date=?, updated_at=? WHERE id=?`,
		rec.Type, rec.AccountID, rec.ToAccountID, rec.CategoryID, rec.Amount, toAmt,
		rec.Description, rec.NextRunDate, now, pendingID)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(rec.Tags))
	for _, t := range rec.Tags {
		ids = append(ids, t.ID)
	}
	return s.Txs.saveTags(ctx, pendingID, ids)
}

func (s RecurringStore) pendingID(ctx context.Context, recurringID int64) (int64, error) {
	var id sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		`SELECT id FROM transactions WHERE recurring_transaction_id = ? AND status = 'pending' LIMIT 1`,
		recurringID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id.Int64, err
}

func (s RecurringStore) calculateToAmount(ctx context.Context, rec *Recurring) (float64, error) {
	if rec.ToAccountID == nil {
		return rec.Amount, nil
	}
	accts := Accounts{DB: s.DB}
	from, err := accts.ByID(ctx, rec.AccountID)
	if err != nil || from == nil || from.Currency == nil {
		return rec.Amount, err
	}
	to, err := accts.ByID(ctx, *rec.ToAccountID)
	if err != nil || to == nil || to.Currency == nil {
		return rec.Amount, err
	}
	return Convert(rec.Amount, *from.Currency, *to.Currency), nil
}

func (s RecurringStore) saveTags(ctx context.Context, id int64, tagIDs []int64) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM recurring_transaction_tag WHERE recurring_transaction_id = ?`, id); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO recurring_transaction_tag (recurring_transaction_id, tag_id) VALUES (?,?)`, id, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (s RecurringStore) list(ctx context.Context, where string, args ...any) ([]Recurring, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, type, account_id, to_account_id, category_id, amount, to_amount, description,
			frequency, interval, day_of_week, day_of_month, start_date, end_date,
			next_run_date, last_run_date, is_active
		FROM recurring_transactions `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recurring
	for rows.Next() {
		r, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	accts := Accounts{DB: s.DB}
	cats := Categories{DB: s.DB}
	for i := range out {
		if a, _ := accts.ByID(ctx, out[i].AccountID); a != nil {
			out[i].Account = a
		}
		if out[i].ToAccountID != nil {
			if a, _ := accts.ByID(ctx, *out[i].ToAccountID); a != nil {
				out[i].ToAccount = a
			}
		}
		if out[i].CategoryID != nil {
			if c, _ := cats.ByID(ctx, *out[i].CategoryID); c != nil {
				out[i].Category = c
			}
		}
		out[i].Tags, _ = s.tags(ctx, out[i].ID)
	}
	return out, nil
}

func (s RecurringStore) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT tags.id, tags.name, tags.created_at, 0 FROM tags
		JOIN recurring_transaction_tag rt ON rt.tag_id = tags.id
		WHERE rt.recurring_transaction_id = ?`, id)
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

func (r Recurring) withinSchedule() bool {
	if !r.IsActive {
		return false
	}
	if r.EndDate == nil || *r.EndDate == "" {
		return true
	}
	return r.NextRunDate <= *r.EndDate
}

func (r Recurring) calculateNextRunDate() string {
	cur, err := time.ParseInLocation("2006-01-02", r.NextRunDate, time.UTC)
	if err != nil {
		return r.NextRunDate
	}
	interval := r.Interval
	if interval < 1 {
		interval = 1
	}
	var next time.Time
	switch r.Frequency {
	case "daily":
		next = cur.AddDate(0, 0, interval)
	case "weekly":
		next = cur.AddDate(0, 0, 7*interval)
		if r.DayOfWeek != nil && int(next.Weekday()) != *r.DayOfWeek {
			next = nextWeekday(next, time.Weekday(*r.DayOfWeek))
		}
	case "monthly":
		next = addMonthsNoOverflow(cur, interval)
		if r.DayOfMonth != nil {
			day := *r.DayOfMonth
			last := daysInMonth(next)
			if day > last {
				day = last
			}
			next = time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, time.UTC)
		}
	case "yearly":
		next = cur.AddDate(interval, 0, 0)
	default:
		next = cur.AddDate(0, 0, interval)
	}
	return next.Format("2006-01-02")
}

func nextWeekday(t time.Time, want time.Weekday) time.Time {
	delta := (int(want) - int(t.Weekday()) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	return t.AddDate(0, 0, delta)
}

func addMonthsNoOverflow(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := daysInMonth(first)
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, 0, 0, 0, 0, time.UTC)
}

func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func scanRecurring(row interface{ Scan(...any) error }) (Recurring, error) {
	var r Recurring
	var toAcc, cat sql.NullInt64
	var toAmt sql.NullFloat64
	var desc, end, last sql.NullString
	var dow, dom sql.NullInt64
	var active int
	err := row.Scan(&r.ID, &r.Type, &r.AccountID, &toAcc, &cat, &r.Amount, &toAmt, &desc,
		&r.Frequency, &r.Interval, &dow, &dom, &r.StartDate, &end, &r.NextRunDate, &last, &active)
	if toAcc.Valid {
		r.ToAccountID = &toAcc.Int64
	}
	if cat.Valid {
		r.CategoryID = &cat.Int64
	}
	if toAmt.Valid {
		r.ToAmount = &toAmt.Float64
	}
	if desc.Valid {
		r.Description = &desc.String
	}
	if dow.Valid {
		v := int(dow.Int64)
		r.DayOfWeek = &v
	}
	if dom.Valid {
		v := int(dom.Int64)
		r.DayOfMonth = &v
	}
	if end.Valid {
		r.EndDate = &end.String
	}
	if last.Valid {
		r.LastRunDate = &last.String
	}
	r.IsActive = active != 0
	return r, err
}
