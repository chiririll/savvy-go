package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// appLocation is the timezone used for calendar-day concepts across this
// package: "today", and whether a date is in the future. Configured once at
// startup from the TZ env var (see config.Config.Location); defaults to UTC.
// A raw time.Now().UTC() day boundary would disagree with a client whose
// local day has already rolled over, wrongly flagging same-day dates as future.
var appLocation = time.UTC

// SetLocation configures appLocation. Call once during server startup.
func SetLocation(loc *time.Location) {
	if loc != nil {
		appLocation = loc
	}
}

// TxItem is a line of a transaction, priced in the currency of its account.
type TxItem struct {
	ID           int64
	Name         string
	Quantity     decimal.Decimal
	PricePerUnit money.Money
	TotalPrice   money.Money
}

// TxItemInput is a line as entered. TotalPrice is Quantity times PricePerUnit
// when zero.
type TxItemInput struct {
	Name         string
	Quantity     decimal.Decimal
	PricePerUnit decimal.Decimal
	TotalPrice   decimal.Decimal
}

type Transaction struct {
	ID          int64
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      money.Money  // in the currency of Account
	ToAmount    *money.Money // in the currency of ToAccount
	Description *string
	Date        *string
	Status      string
	RecurringID *int64
	CreatedAt   *time.Time
	Account     *Account
	ToAccount   *Account
	Category    *Category
	Items       []TxItem
	Tags        []Tag
}

type TxInput struct {
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      decimal.Decimal
	ToAmount    *decimal.Decimal
	Description *string
	Date        *string
	Status      *string
	RecurringID *int64
	TagIDs      []int64
	Items       []TxItemInput
}

type Transactions struct{ DB *sql.DB }

func (s Transactions) Create(ctx context.Context, in TxInput) (*Transaction, error) {
	status := "confirmed"
	if in.Status != nil {
		status = *in.Status
	} else if in.Date == nil || isFuture(*in.Date) {
		status = "pending"
	}
	if status == "confirmed" {
		if in.Date == nil {
			return nil, fmt.Errorf("date required")
		}
		if isFuture(*in.Date) {
			return nil, fmt.Errorf("future")
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	amount, toAmount, err := s.amounts(ctx, in)
	if err != nil {
		return nil, err
	}
	res, err := db.Q(s.DB).InsertTransaction(ctx, sqlc.InsertTransactionParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: amount.Minor(), ToAmount: money.ToNullMinor(toAmount),
		Description: db.NullString(in.Description),
		Date:        db.NullString(in.Date), Status: status, RecurringTransactionID: db.NullInt64(in.RecurringID),
		CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if err := s.saveItems(ctx, id, in.Items, amount.Unit()); err != nil {
		return nil, err
	}
	if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Update(ctx context.Context, id int64, in TxInput) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status == "skipped" || cur.RecurringID != nil {
		return nil, fmt.Errorf("cannot edit")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	amount, toAmount, err := s.amounts(ctx, in)
	if err != nil {
		return nil, err
	}
	err = db.Q(s.DB).UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: db.NullInt64(in.ToAccountID),
		CategoryID: db.NullInt64(in.CategoryID), Amount: amount.Minor(), ToAmount: money.ToNullMinor(toAmount),
		Description: db.NullString(in.Description),
		Date:        db.NullString(in.Date), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	if in.Items != nil {
		_ = db.Q(s.DB).DeleteTransactionItems(ctx, id)
		if err := s.saveItems(ctx, id, in.Items, amount.Unit()); err != nil {
			return nil, err
		}
	}
	if in.TagIDs != nil {
		if err := s.saveTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if cur.RecurringID != nil {
		return fmt.Errorf("cannot delete recurring")
	}
	return db.Q(s.DB).DeleteTransaction(ctx, id)
}

func (s Transactions) Confirm(ctx context.Context, id int64, date string) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status != "pending" {
		return nil, fmt.Errorf("not pending")
	}
	if date == "" && cur.Date != nil {
		date = *cur.Date
	}
	if date == "" {
		return nil, fmt.Errorf("date required")
	}
	if isFuture(date) {
		return nil, fmt.Errorf("future")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).ConfirmTransaction(ctx, sqlc.ConfirmTransactionParams{Date: db.NS(date), UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Skip(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.Status != "pending" || cur.RecurringID == nil {
		return nil, fmt.Errorf("cannot skip")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = db.Q(s.DB).SkipTransaction(ctx, sqlc.SkipTransactionParams{UpdatedAt: db.NS(now), ID: id})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Duplicate(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if cur.RecurringID != nil {
		return nil, fmt.Errorf("cannot duplicate")
	}
	today := time.Now().In(appLocation).Format("2006-01-02")
	in := TxInput{
		Type: cur.Type, AccountID: cur.AccountID, ToAccountID: cur.ToAccountID,
		CategoryID: cur.CategoryID, Amount: cur.Amount.Decimal(),
		Description: cur.Description, Date: &today,
	}
	if cur.ToAmount != nil {
		d := cur.ToAmount.Decimal()
		in.ToAmount = &d
	}
	for _, it := range cur.Items {
		in.Items = append(in.Items, TxItemInput{
			Name: it.Name, Quantity: it.Quantity,
			PricePerUnit: it.PricePerUnit.Decimal(), TotalPrice: it.TotalPrice.Decimal(),
		})
	}
	for _, t := range cur.Tags {
		in.TagIDs = append(in.TagIDs, t.ID)
	}
	st := "confirmed"
	in.Status = &st
	return s.Create(ctx, in)
}

func (s Transactions) ByID(ctx context.Context, id int64) (*Transaction, error) {
	list, err := s.list(ctx, sqlc.ListTransactionsParams{ID: db.NI(id), Limit: 1, Offset: 0})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Transactions) Filtered(ctx context.Context, f filter.TxFilter, page, perPage int) ([]Transaction, int, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	total, err := filter.CountTransactions(ctx, s.DB, f)
	if err != nil {
		return nil, 0, err
	}
	rows, err := filter.ListTransactions(ctx, s.DB, f, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	return s.hydrate(ctx, rows), int(total), nil
}

// TransactionSummary totals transactions in the base currency (nil when there
// is none), rounded to its decimals.
type TransactionSummary struct {
	Income   money.Money
	Expense  money.Money
	Count    int
	Currency *Currency
}

func (s Transactions) Summary(ctx context.Context, pendingOnly bool) TransactionSummary {
	status := "confirmed"
	if pendingOnly {
		status = "pending"
	}
	base, _ := (Currencies{DB: s.DB}).Base(ctx)
	out := TransactionSummary{Currency: base}
	var unit money.Unit // no base currency means no transactions either
	if base != nil {
		unit = base.Unit()
	}
	out.Income, out.Expense = money.Zero(unit), money.Zero(unit)
	rows, err := db.Q(s.DB).ListTransactionSummaryRows(ctx, status)
	if err != nil {
		return out
	}
	// Sum in base major units and round once: rounding each row would drift.
	income, expense := decimal.Zero, decimal.Zero
	for _, r := range rows {
		amount := decimal.New(r.Amount, -int32(r.Decimals))
		if r.IsBase == 0 && !r.Rate.IsZero() {
			amount = amount.Mul(r.Rate)
		}
		if r.Type == "income" {
			income = income.Add(amount)
		} else {
			expense = expense.Add(amount)
		}
		out.Count++
	}
	out.Income, out.Expense = money.FromDecimal(income, unit), money.FromDecimal(expense, unit)
	return out
}

func (s Transactions) list(ctx context.Context, arg sqlc.ListTransactionsParams) ([]Transaction, error) {
	rows, err := db.Q(s.DB).ListTransactions(ctx, arg)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, rows), nil
}

func (s Transactions) hydrate(ctx context.Context, rows []sqlc.ListTransactionsRow) []Transaction {
	out := make([]Transaction, 0, len(rows))
	for _, r := range rows {
		out = append(out, txFromRow(r))
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
		out[i].Items, _ = s.items(ctx, out[i].ID, out[i].Amount.Unit())
		out[i].Tags, _ = s.tags(ctx, out[i].ID)
	}
	return out
}

func (s Transactions) items(ctx context.Context, id int64, unit money.Unit) ([]TxItem, error) {
	rows, err := db.Q(s.DB).ListTransactionItems(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]TxItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, TxItem{ID: r.ID, Name: r.Name, Quantity: r.Quantity, PricePerUnit: money.New(r.PricePerUnit, unit), TotalPrice: money.New(r.TotalPrice, unit)})
	}
	return out, nil
}

func (s Transactions) tags(ctx context.Context, id int64) ([]Tag, error) {
	rows, err := db.Q(s.DB).ListTransactionTags(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, tagFromList(r.ID, r.Name, r.CreatedAt, r.TransactionsCount))
	}
	return out, nil
}

func (s Transactions) saveItems(ctx context.Context, txID int64, items []TxItemInput, unit money.Unit) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, it := range items {
		if it.TotalPrice.IsZero() {
			it.TotalPrice = it.Quantity.Mul(it.PricePerUnit)
		}
		if err := db.Q(s.DB).InsertTransactionItem(ctx, sqlc.InsertTransactionItemParams{
			TransactionID: txID, Name: it.Name, Quantity: it.Quantity,
			PricePerUnit: money.FromDecimal(it.PricePerUnit, unit).Minor(),
			TotalPrice:   money.FromDecimal(it.TotalPrice, unit).Minor(), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s Transactions) saveTags(ctx context.Context, txID int64, ids []int64) error {
	_ = db.Q(s.DB).DeleteTransactionTags(ctx, txID)
	for _, id := range ids {
		if err := db.Q(s.DB).InsertTransactionTag(ctx, sqlc.InsertTransactionTagParams{TransactionID: txID, TagID: id}); err != nil {
			return err
		}
	}
	return nil
}

// unitsFor returns the unit of the account and, when set, of the to-account;
// the to-account falls back to the account's unit.
func (s Transactions) unitsFor(ctx context.Context, accountID int64, toAccountID *int64) (money.Unit, money.Unit, error) {
	accts := Accounts{DB: s.DB}
	unit, err := accts.Unit(ctx, accountID)
	if err != nil {
		return money.Unit{}, money.Unit{}, err
	}
	if toAccountID == nil {
		return unit, unit, nil
	}
	toUnit, err := accts.Unit(ctx, *toAccountID)
	if err != nil {
		return money.Unit{}, money.Unit{}, err
	}
	return unit, toUnit, nil
}

// amounts turns the entered amounts into money in the currencies of the
// accounts. A transfer without a destination amount delivers the amount
// converted at the current rates, never the same number reinterpreted in
// another currency.
func (s Transactions) amounts(ctx context.Context, in TxInput) (amount money.Money, toAmount *money.Money, err error) {
	unit, toUnit, err := s.unitsFor(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return money.Money{}, nil, err
	}
	amount = money.FromDecimal(in.Amount, unit)
	toAmount = money.FromNullDecimal(in.ToAmount, toUnit)
	if toAmount != nil || in.Type != "transfer" || in.ToAccountID == nil {
		return amount, toAmount, nil
	}
	if unit == toUnit {
		return amount, &amount, nil
	}
	accts := Accounts{DB: s.DB}
	from, err := accts.ByID(ctx, in.AccountID)
	if err != nil || from == nil {
		return money.Money{}, nil, ErrUnknownAccount
	}
	to, err := accts.ByID(ctx, *in.ToAccountID)
	if err != nil || to == nil {
		return money.Money{}, nil, ErrUnknownAccount
	}
	converted := Convert(amount, *from.Currency, *to.Currency)
	return amount, &converted, nil
}

func txFromRow(r sqlc.ListTransactionsRow) Transaction {
	t := Transaction{
		ID: r.ID, Type: r.Type, AccountID: r.AccountID, Status: r.Status,
		Amount:   money.New(r.Amount, money.Unit{ID: r.CurrencyID, Decimals: int(r.Decimals)}),
		ToAmount: money.FromNullMinor(r.ToAmount, money.Unit{ID: r.ToCurrencyID, Decimals: int(r.ToDecimals)}),
	}
	if r.ToAccountID.Valid {
		t.ToAccountID = &r.ToAccountID.Int64
	}
	if r.CategoryID.Valid {
		t.CategoryID = &r.CategoryID.Int64
	}
	if r.Description.Valid {
		t.Description = &r.Description.String
	}
	if r.Date.Valid {
		t.Date = &r.Date.String
	}
	if r.RecurringTransactionID.Valid {
		t.RecurringID = &r.RecurringTransactionID.Int64
	}
	if tm, ok := parseNullTime(r.CreatedAt); ok {
		t.CreatedAt = &tm
	}
	return t
}

// isFuture compares plain "YYYY-MM-DD" calendar dates as strings, using
// appLocation's day as "today". A time.Time-based comparison would parse the
// date at UTC midnight while "now" is truncated separately, so a client whose
// local day has already rolled over (e.g. UTC+3 just after midnight) could
// have a same-day date wrongly rejected as future. Only the first 10 chars
// are used so legacy rows stored as "YYYY-MM-DD HH:MM:SS" still compare correctly.
func isFuture(date string) bool {
	if len(date) < 10 {
		return false
	}
	today := time.Now().In(appLocation).Format("2006-01-02")
	return date[:10] > today
}
