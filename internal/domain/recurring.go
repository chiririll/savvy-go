package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
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
	return s.list(ctx, sqlc.ListRecurringParams{Limit: 1_000_000, Offset: 0})
}

func (s RecurringStore) Upcoming(ctx context.Context, limit int) ([]Recurring, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.list(ctx, sqlc.ListRecurringParams{ActiveOnly: db.Flag(true), Limit: int64(limit), Offset: 0})
}

func (s RecurringStore) ByID(ctx context.Context, id int64) (*Recurring, error) {
	list, err := s.list(ctx, sqlc.ListRecurringParams{ID: db.NI(id), Limit: 1, Offset: 0})
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
	res, err := db.Q(s.DB).InsertRecurring(ctx, sqlc.InsertRecurringParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: in.Amount, ToAmount: db.NullFloat64(in.ToAmount),
		Description: db.NullString(in.Description), Frequency: in.Frequency, Interval: int64(in.Interval),
		DayOfWeek: db.NullInt(in.DayOfWeek), DayOfMonth: db.NullInt(in.DayOfMonth),
		StartDate: in.StartDate, EndDate: db.NullString(in.EndDate), NextRunDate: in.StartDate,
		IsActive: db.BoolInt(active), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
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
	err = db.Q(s.DB).UpdateRecurring(ctx, sqlc.UpdateRecurringParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: in.Amount, ToAmount: db.NullFloat64(in.ToAmount),
		Description: db.NullString(in.Description), Frequency: in.Frequency, Interval: int64(in.Interval),
		DayOfWeek: db.NullInt(in.DayOfWeek), DayOfMonth: db.NullInt(in.DayOfMonth),
		StartDate: in.StartDate, EndDate: db.NullString(in.EndDate), NextRunDate: next,
		IsActive: db.BoolInt(active), UpdatedAt: db.NS(now), ID: id,
	})
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
	if err := db.Q(s.DB).DeletePendingForRecurring(ctx, db.NI(id)); err != nil {
		return err
	}
	return db.Q(s.DB).DeleteRecurring(ctx, id)
}

func (s RecurringStore) AdvanceAfterOccurrence(ctx context.Context, id int64) error {
	rec, err := s.ByID(ctx, id)
	if err != nil || rec == nil {
		return err
	}
	next := rec.calculateNextRunDate()
	now := time.Now().UTC()
	err = db.Q(s.DB).AdvanceRecurring(ctx, sqlc.AdvanceRecurringParams{
		LastRunDate: db.NS(now.Format("2006-01-02")), NextRunDate: next,
		UpdatedAt: db.NS(now.Format(time.RFC3339)), ID: id,
	})
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
	list, err := s.list(ctx, sqlc.ListRecurringParams{ActiveOnly: db.Flag(true), Limit: 1_000_000, Offset: 0})
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
	var toAmtN sql.NullFloat64
	switch v := toAmt.(type) {
	case float64:
		toAmtN = sql.NullFloat64{Float64: v, Valid: true}
	case *float64:
		toAmtN = db.NullFloat64(v)
	}
	err = db.Q(s.DB).UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: rec.Type, AccountID: rec.AccountID, ToAccountID: db.NullInt64(rec.ToAccountID),
		CategoryID: db.NullInt64(rec.CategoryID), Amount: rec.Amount, ToAmount: toAmtN,
		Description: db.NullString(rec.Description), Date: db.NS(rec.NextRunDate),
		UpdatedAt: db.NS(now), ID: pendingID,
	})
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
	id, err := db.Q(s.DB).GetPendingRecurringTx(ctx, db.NI(recurringID))
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
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
	if err := db.Q(s.DB).DeleteRecurringTags(ctx, id); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := db.Q(s.DB).InsertRecurringTag(ctx, sqlc.InsertRecurringTagParams{RecurringTransactionID: id, TagID: tagID}); err != nil {
			return err
		}
	}
	return nil
}

func (s RecurringStore) list(ctx context.Context, arg sqlc.ListRecurringParams) ([]Recurring, error) {
	rows, err := db.Q(s.DB).ListRecurring(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Recurring, 0, len(rows))
	for _, r := range rows {
		out = append(out, recurringFromRow(r))
	}
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
	rows, err := db.Q(s.DB).ListRecurringTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, tagFromList(r.ID, r.Name, r.CreatedAt, r.TransactionsCount))
	}
	return out, nil
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

func recurringFromRow(row sqlc.ListRecurringRow) Recurring {
	r := Recurring{
		ID: row.ID, Type: row.Type, AccountID: row.AccountID, Amount: row.Amount,
		Frequency: row.Frequency, Interval: int(row.Interval), StartDate: row.StartDate,
		NextRunDate: row.NextRunDate, IsActive: row.IsActive != 0,
	}
	if row.ToAccountID.Valid {
		r.ToAccountID = &row.ToAccountID.Int64
	}
	if row.CategoryID.Valid {
		r.CategoryID = &row.CategoryID.Int64
	}
	if row.ToAmount.Valid {
		r.ToAmount = &row.ToAmount.Float64
	}
	if row.Description.Valid {
		r.Description = &row.Description.String
	}
	if row.DayOfWeek.Valid {
		v := int(row.DayOfWeek.Int64)
		r.DayOfWeek = &v
	}
	if row.DayOfMonth.Valid {
		v := int(row.DayOfMonth.Int64)
		r.DayOfMonth = &v
	}
	if row.EndDate.Valid {
		r.EndDate = &row.EndDate.String
	}
	if row.LastRunDate.Valid {
		r.LastRunDate = &row.LastRunDate.String
	}
	return r
}
