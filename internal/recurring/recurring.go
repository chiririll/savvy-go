package recurring

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/recurring"
	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/platform/calendar"
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

func (s Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

func (s Service) All(ctx context.Context) ([]Recurring, error) {
	return s.list(ctx, false, 1_000_000)
}

func (s Service) Upcoming(ctx context.Context, limit int) ([]Recurring, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.list(ctx, true, limit)
}

func (s Service) ByID(ctx context.Context, id int64) (*Recurring, error) {
	r, err := s.repo().ByID(ctx, id)
	if r == nil || err != nil {
		return nil, err
	}
	list, err := s.hydrate(ctx, []repo.Row{*r})
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
	unit, toUnit, err := s.Txs.Units(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return nil, err
	}
	amount, toAmount, err := recurringAmounts(in, unit, toUnit)
	if err != nil {
		return nil, err
	}
	id, err := s.repo().Insert(ctx, repo.Fields{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: in.ToAccountID, CategoryID: in.CategoryID,
		Amount: amount, ToAmount: toAmount, IsEstimated: estimated, Description: in.Description,
		Frequency: in.Frequency, Interval: in.Interval, DayOfWeek: in.DayOfWeek, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, EndDate: in.EndDate, NextRunDate: in.StartDate, IsActive: active,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := s.repo().SetTags(ctx, id, in.TagIDs); err != nil {
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

// fillFromCurrent keeps the stored value for every required field in left empty.
func fillFromCurrent(in *Input, cur *Recurring) {
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
}

func (s Service) Update(ctx context.Context, id int64, in Input) (*Recurring, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	fillFromCurrent(&in, cur)
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
	unit, toUnit, err := s.Txs.Units(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return nil, err
	}
	amount, toAmount, err := recurringAmounts(in, unit, toUnit)
	if err != nil {
		return nil, err
	}
	err = s.repo().Update(ctx, id, repo.Fields{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: in.ToAccountID, CategoryID: in.CategoryID,
		Amount: amount, ToAmount: toAmount, IsEstimated: estimated, Description: in.Description,
		Frequency: in.Frequency, Interval: in.Interval, DayOfWeek: in.DayOfWeek, DayOfMonth: in.DayOfMonth,
		StartDate: in.StartDate, EndDate: in.EndDate, NextRunDate: next, IsActive: active,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if in.HasTagIDs {
		if err := s.repo().SetTags(ctx, id, in.TagIDs); err != nil {
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
	return s.repo().Delete(ctx, id)
}

func (s Service) AdvanceAfterOccurrence(ctx context.Context, id int64) error {
	rec, err := s.ByID(ctx, id)
	if err != nil || rec == nil {
		return err
	}
	next := rec.calculateNextRunDate()
	now := time.Now().UTC()
	err = s.repo().Advance(ctx, id, now.Format("2006-01-02"), next, now)
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
	list, err := s.list(ctx, true, 1_000_000)
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
	ids := make([]int64, 0, len(rec.Tags))
	for _, t := range rec.Tags {
		ids = append(ids, t.ID)
	}
	return s.Txs.Replan(ctx, pendingID, ledger.Plan{
		Type: rec.Type, AccountID: rec.AccountID, ToAccountID: rec.ToAccountID, CategoryID: rec.CategoryID,
		Amount: rec.Amount, ToAmount: toAmt, IsEstimated: rec.IsEstimated, Description: rec.Description,
		Date: rec.NextRunDate, TagIDs: ids,
	})
}

func (s Service) pendingID(ctx context.Context, recurringID int64) (int64, error) {
	return s.repo().PendingTransactionID(ctx, recurringID)
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

func (s Service) list(ctx context.Context, activeOnly bool, limit int) ([]Recurring, error) {
	rows, err := s.repo().List(ctx, activeOnly, limit)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, rows)
}

// hydrate turns rows into recurrences with their accounts, category and tags,
// loading each distinct account and category once.
func (s Service) hydrate(ctx context.Context, rows []repo.Row) ([]Recurring, error) {
	out := make([]Recurring, 0, len(rows))
	rel := ledger.NewRelated(ctx, s.DB)
	for _, r := range rows {
		rec := recurringFrom(r)
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
	rows, err := s.repo().Tags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]ledger.Tag, len(rows))
	for i, r := range rows {
		out[i] = ledger.Tag{ID: r.ID, Name: r.Name, TransactionsCount: r.TransactionsCount, CreatedAt: r.CreatedAt}
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

func recurringFrom(row repo.Row) Recurring {
	return Recurring{
		ID: row.ID, Type: row.Type, AccountID: row.AccountID, ToAccountID: row.ToAccountID, CategoryID: row.CategoryID,
		Amount: row.Amount, ToAmount: row.ToAmount, IsEstimated: row.IsEstimated, Description: row.Description,
		Frequency: row.Frequency, Interval: row.Interval, DayOfWeek: row.DayOfWeek, DayOfMonth: row.DayOfMonth,
		StartDate: row.StartDate, EndDate: row.EndDate, NextRunDate: row.NextRunDate, LastRunDate: row.LastRunDate,
		IsActive: row.IsActive,
	}
}
