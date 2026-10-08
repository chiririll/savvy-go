package recurring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/calendar"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

type Recurring struct {
	ID          int64
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      money.Money  // in the currency of Account
	ToAmount    *money.Money // in the currency of ToAccount
	IsEstimated bool         // the amount is a rough guess; each occurrence is confirmed with the real one
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
	Account     *ledger.Account
	ToAccount   *ledger.Account
	Category    *ledger.Category
	Tags        []ledger.Tag
}

type Input struct {
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      decimal.Decimal
	ToAmount    *decimal.Decimal
	IsEstimated *bool
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

type Service struct {
	DB  store.DB
	Txs ledger.Transactions
}

func (s Service) All(ctx context.Context) ([]Recurring, error) {
	return s.list(ctx, sqlc.ListRecurringParams{Limit: 1_000_000, Offset: 0})
}

func (s Service) Upcoming(ctx context.Context, limit int) ([]Recurring, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.list(ctx, sqlc.ListRecurringParams{ActiveOnly: db.Flag(true), Limit: int64(limit), Offset: 0})
}

func (s Service) ByID(ctx context.Context, id int64) (*Recurring, error) {
	r, err := db.Q(s.DB).GetRecurring(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	list, err := s.hydrate(ctx, []sqlc.ListRecurringRow{sqlc.ListRecurringRow(r)})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (s Service) Create(ctx context.Context, in Input) (*Recurring, error) {
	if in.Interval < 1 {
		in.Interval = 1
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	estimated := in.IsEstimated != nil && *in.IsEstimated
	now := time.Now().UTC().Format(time.RFC3339)
	unit, toUnit, err := s.Txs.Units(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return nil, err
	}
	amount, toAmount, err := recurringAmounts(in, unit, toUnit)
	if err != nil {
		return nil, err
	}
	res, err := db.Q(s.DB).InsertRecurring(ctx, sqlc.InsertRecurringParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: amount.Minor(),
		ToAmount: db.NullMoney(toAmount), IsEstimated: db.BoolInt(estimated),
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

func (s Service) Update(ctx context.Context, id int64, in Input) (*Recurring, error) {
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
	if in.Amount.IsZero() {
		in.Amount = cur.Amount.Decimal()
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
	estimated := cur.IsEstimated
	if in.IsEstimated != nil {
		estimated = *in.IsEstimated
	}
	now := time.Now().UTC().Format(time.RFC3339)
	unit, toUnit, err := s.Txs.Units(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return nil, err
	}
	amount, toAmount, err := recurringAmounts(in, unit, toUnit)
	if err != nil {
		return nil, err
	}
	err = db.Q(s.DB).UpdateRecurring(ctx, sqlc.UpdateRecurringParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: amount.Minor(),
		ToAmount: db.NullMoney(toAmount), IsEstimated: db.BoolInt(estimated),
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

func (s Service) Delete(ctx context.Context, id int64) error {
	if err := db.Q(s.DB).DeletePendingForRecurring(ctx, db.NI(id)); err != nil {
		return err
	}
	return db.Q(s.DB).DeleteRecurring(ctx, id)
}

func (s Service) AdvanceAfterOccurrence(ctx context.Context, id int64) error {
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
func (s Service) EnsureUpcoming(ctx context.Context) error {
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

// recurringAmounts checks the amounts entered for a recurrence against the
// range and rounds them into the currencies of its accounts.
func recurringAmounts(in Input, unit, toUnit money.Unit) (amount money.Money, toAmount *money.Money, err error) {
	if amount, err = money.FromInput(in.Amount, unit); err != nil {
		return money.Money{}, nil, err
	}
	toAmount, err = money.FromNullInput(in.ToAmount, toUnit)
	return amount, toAmount, err
}

func (s Service) createPending(ctx context.Context, rec *Recurring) (*ledger.Transaction, error) {
	var toAmt *decimal.Decimal
	if rec.Type == "transfer" {
		v, err := s.transferAmount(ctx, rec)
		if err != nil {
			return nil, err
		}
		d := v.Decimal()
		toAmt = &d
	}
	status := "pending"
	date := rec.NextRunDate
	in := ledger.TxInput{
		Type: rec.Type, AccountID: rec.AccountID, ToAccountID: rec.ToAccountID,
		CategoryID: rec.CategoryID, Amount: rec.Amount.Decimal(), ToAmount: toAmt, IsEstimated: rec.IsEstimated,
		Description: rec.Description, Date: &date, Status: &status, RecurringID: &rec.ID,
	}
	for _, t := range rec.Tags {
		in.TagIDs = append(in.TagIDs, t.ID)
	}
	return s.Txs.Create(ctx, in)
}

func (s Service) syncOpenPending(ctx context.Context, rec *Recurring) error {
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
	var toAmt *money.Money
	if rec.Type == "transfer" {
		v, err := s.transferAmount(ctx, rec)
		if err != nil {
			return err
		}
		toAmt = &v
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: rec.Type, AccountID: rec.AccountID, ToAccountID: db.NullInt64(rec.ToAccountID),
		CategoryID: db.NullInt64(rec.CategoryID), Amount: rec.Amount.Minor(), ToAmount: db.NullMoney(toAmt),
		IsEstimated: db.BoolInt(rec.IsEstimated), Description: db.NullString(rec.Description), Date: db.NS(rec.NextRunDate),
		UpdatedAt: db.NS(now), ID: pendingID,
	})
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(rec.Tags))
	for _, t := range rec.Tags {
		ids = append(ids, t.ID)
	}
	return s.Txs.SetTags(ctx, pendingID, ids)
}

func (s Service) pendingID(ctx context.Context, recurringID int64) (int64, error) {
	id, err := db.Q(s.DB).GetPendingRecurringTx(ctx, db.NI(recurringID))
	if err == db.ErrNotFound {
		return 0, nil
	}
	return id, err
}

// transferAmount is what a transfer recurrence delivers to the destination
// account: its own amount when set, else Amount converted between the accounts'
// currencies at the current rates.
func (s Service) transferAmount(ctx context.Context, rec *Recurring) (money.Money, error) {
	if rec.ToAmount != nil {
		return *rec.ToAmount, nil
	}
	if rec.ToAccountID == nil {
		return rec.Amount, nil
	}
	accts := ledger.Accounts{DB: s.DB}
	from, err := accts.ByID(ctx, rec.AccountID)
	if err != nil || from == nil {
		return rec.Amount, err
	}
	to, err := accts.ByID(ctx, *rec.ToAccountID)
	if err != nil || to == nil {
		return rec.Amount, err
	}
	return ledger.Convert(rec.Amount, *from.Currency, *to.Currency), nil
}

func (s Service) saveTags(ctx context.Context, id int64, tagIDs []int64) error {
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

func (s Service) list(ctx context.Context, arg sqlc.ListRecurringParams) ([]Recurring, error) {
	rows, err := db.Q(s.DB).ListRecurring(ctx, arg)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, rows)
}

// hydrate turns rows into recurrences with their accounts, category and tags,
// loading each distinct account and category once.
func (s Service) hydrate(ctx context.Context, rows []sqlc.ListRecurringRow) ([]Recurring, error) {
	out := make([]Recurring, 0, len(rows))
	rel := ledger.NewRelated(ctx, s.DB)
	for _, r := range rows {
		rec := recurringFromRow(r)
		rec.Account = rel.Account(rec.AccountID)
		rec.ToAccount = rel.OptionalAccount(rec.ToAccountID)
		rec.Category = rel.Category(rec.CategoryID)
		tags, err := s.tags(ctx, rec.ID)
		if err != nil {
			return nil, err
		}
		rec.Tags = tags
		out = append(out, rec)
	}
	return out, nil
}

func (s Service) tags(ctx context.Context, id int64) ([]ledger.Tag, error) {
	rows, err := db.Q(s.DB).ListRecurringTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]ledger.Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, ledger.TagFromRow(r.ID, r.Name, r.CreatedAt, r.TransactionsCount))
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
	dateOnly := r.NextRunDate
	if len(dateOnly) > 10 {
		dateOnly = dateOnly[:10]
	}
	cur, err := time.ParseInLocation("2006-01-02", dateOnly, time.UTC)
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
		next = calendar.AddMonths(cur, interval)
		if r.DayOfMonth != nil {
			day := *r.DayOfMonth
			last := calendar.DaysInMonth(next)
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

func recurringFromRow(row sqlc.ListRecurringRow) Recurring {
	r := Recurring{
		ID: row.ID, Type: row.Type, AccountID: row.AccountID,
		Amount:    money.New(row.Amount, money.Unit{ID: row.CurrencyID, Decimals: int(row.Decimals)}),
		ToAmount:  db.MoneyFromNull(row.ToAmount, money.Unit{ID: row.ToCurrencyID, Decimals: int(row.ToDecimals)}),
		Frequency: row.Frequency, Interval: int(row.Interval), StartDate: row.StartDate,
		NextRunDate: row.NextRunDate, IsActive: row.IsActive != 0, IsEstimated: row.IsEstimated != 0,
	}
	if row.ToAccountID.Valid {
		r.ToAccountID = &row.ToAccountID.Int64
	}
	if row.CategoryID.Valid {
		r.CategoryID = &row.CategoryID.Int64
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
