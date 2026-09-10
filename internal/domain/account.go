package domain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

type Account struct {
	ID             int64
	Name           string
	Type           string
	CurrencyID     int64
	InitialBalance float64
	IsActive       bool
	SortOrder      int
	DebtType       *string
	TargetAmount   *float64
	DueDate        *string
	IsPaidOff      bool
	Counterparty   *string
	DebtDesc       *string
	CreatedAt      *time.Time
	Currency       *Currency
	Balance        float64
}

func (a Account) JSON() map[string]any {
	var created any
	if a.CreatedAt != nil {
		created = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	var cur any
	if a.Currency != nil {
		cur = a.Currency.JSON()
	}
	return map[string]any{
		"id":             a.ID,
		"name":           a.Name,
		"type":           a.Type,
		"currencyId":     a.CurrencyID,
		"initialBalance": a.InitialBalance,
		"currentBalance": a.Balance,
		"isActive":       a.IsActive,
		"sortOrder":      a.SortOrder,
		"currency":       cur,
		"createdAt":      created,
	}
}

type Accounts struct{ DB *sql.DB }

func (s Accounts) All(ctx context.Context, onlyActive, excludeDebts bool) ([]Account, error) {
	return s.list(ctx, sqlc.ListAccountsParams{
		OnlyActive:   db.Flag(onlyActive),
		ExcludeDebts: db.Flag(excludeDebts),
	})
}

func (s Accounts) Debts(ctx context.Context, includeCompleted bool) ([]Account, error) {
	return s.list(ctx, sqlc.ListAccountsParams{
		OnlyDebts:  db.Flag(true),
		UnpaidOnly: db.Flag(!includeCompleted),
	})
}

func (s Accounts) ByID(ctx context.Context, id int64) (*Account, error) {
	list, err := s.list(ctx, sqlc.ListAccountsParams{ID: db.NI(id)})
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (s Accounts) Create(ctx context.Context, a Account) (*Account, error) {
	if a.Type == "" {
		a.Type = "cash"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	max, _ := db.Q(s.DB).MaxAccountSortOrder(ctx, db.Flag(a.Type == "debt"))
	a.SortOrder = int(asFloat64(max)) + 1
	res, err := db.Q(s.DB).InsertAccount(ctx, sqlc.InsertAccountParams{
		Name: a.Name, Type: a.Type, DebtType: db.NullString(a.DebtType), CurrencyID: a.CurrencyID,
		InitialBalance: a.InitialBalance, TargetAmount: db.NullFloat64(a.TargetAmount), DueDate: db.NullString(a.DueDate),
		IsPaidOff: db.BoolInt(a.IsPaidOff), Counterparty: db.NullString(a.Counterparty), DebtDescription: db.NullString(a.DebtDesc),
		IsActive: db.BoolInt(a.IsActive), SortOrder: int64(a.SortOrder), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

func (s Accounts) Update(ctx context.Context, id int64, a Account) (*Account, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	err := db.Q(s.DB).UpdateAccount(ctx, sqlc.UpdateAccountParams{
		Name: a.Name, Type: a.Type, CurrencyID: a.CurrencyID, InitialBalance: a.InitialBalance,
		IsActive: db.BoolInt(a.IsActive), DebtType: db.NullString(a.DebtType), TargetAmount: db.NullFloat64(a.TargetAmount),
		DueDate: db.NullString(a.DueDate), IsPaidOff: db.BoolInt(a.IsPaidOff), Counterparty: db.NullString(a.Counterparty),
		DebtDescription: db.NullString(a.DebtDesc), UpdatedAt: db.NS(now), ID: id,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Accounts) Delete(ctx context.Context, id int64) error {
	n, _ := db.Q(s.DB).CountAccountTransactions(ctx, sqlc.CountAccountTransactionsParams{AccountID: id, ToAccountID: db.NI(id)})
	if n > 0 {
		return fmt.Errorf("has transactions")
	}
	return db.Q(s.DB).DeleteAccount(ctx, id)
}

func (s Accounts) Reorder(ctx context.Context, ids []int64) error {
	for i, id := range ids {
		if err := db.Q(s.DB).SetAccountSortOrder(ctx, sqlc.SetAccountSortOrderParams{SortOrder: int64(i), ID: id}); err != nil {
			return err
		}
	}
	return nil
}

func (s Accounts) Summary(ctx context.Context, base *Currency) map[string]any {
	accts, _ := s.All(ctx, true, true)
	total := 0.0
	code := ""
	decimals := 2
	if base != nil {
		code = base.Code
		decimals = base.Decimals
		for _, a := range accts {
			if a.Currency == nil {
				continue
			}
			total += Convert(a.Balance, *a.Currency, *base)
		}
	}
	return map[string]any{
		"total_balance":  roundTo(total, decimals),
		"currency":       nilOr(code),
		"currency_code":  code,
		"decimals":       decimals,
		"accounts_count": len(accts),
	}
}

func (s Accounts) list(ctx context.Context, arg sqlc.ListAccountsParams) ([]Account, error) {
	rows, err := db.Q(s.DB).ListAccounts(ctx, arg)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, accountFromRow(r))
	}
	for i := range out {
		out[i].Balance, _ = s.balance(ctx, out[i], "")
	}
	return out, nil
}

func (s Accounts) balance(ctx context.Context, a Account, asOf string) (float64, error) {
	q := db.Q(s.DB)
	if a.Type == "debt" {
		paid, _ := q.SumDebtPayments(ctx, db.NI(a.ID))
		target := 0.0
		if a.TargetAmount != nil {
			target = *a.TargetAmount
		}
		return target - asFloat64(paid), nil
	}
	asOfArg := db.Narg(asOf)
	income, _ := q.SumAccountIncome(ctx, sqlc.SumAccountIncomeParams{AccountID: a.ID, AsOf: asOfArg})
	expense, _ := q.SumAccountExpense(ctx, sqlc.SumAccountExpenseParams{AccountID: a.ID, AsOf: asOfArg})
	tout, _ := q.SumAccountTransferOut(ctx, sqlc.SumAccountTransferOutParams{AccountID: a.ID, AsOf: asOfArg})
	tin, _ := q.SumAccountTransferIn(ctx, sqlc.SumAccountTransferInParams{ToAccountID: db.NI(a.ID), AsOf: asOfArg})
	dIn, _ := q.SumAccountDebtIn(ctx, sqlc.SumAccountDebtInParams{AccountID: a.ID, AsOf: asOfArg})
	dOut, _ := q.SumAccountDebtOut(ctx, sqlc.SumAccountDebtOutParams{AccountID: a.ID, AsOf: asOfArg})
	return a.InitialBalance + asFloat64(income) - asFloat64(expense) - asFloat64(tout) + asFloat64(tin) + asFloat64(dIn) - asFloat64(dOut), nil
}

func accountFromRow(r sqlc.ListAccountsRow) Account {
	a := Account{
		ID: r.ID, Name: r.Name, Type: r.Type, CurrencyID: r.CurrencyID,
		InitialBalance: r.InitialBalance, IsActive: r.IsActive != 0, SortOrder: int(r.SortOrder),
		IsPaidOff: r.IsPaidOff != 0,
	}
	cur := Currency{ID: r.CurrencyIDJoin, Code: r.Code, Name: r.CurrencyName, Symbol: r.Symbol, Decimals: int(r.Decimals), IsBase: r.IsBase != 0, Rate: r.Rate}
	a.Currency = &cur
	if r.DebtType.Valid {
		a.DebtType = &r.DebtType.String
	}
	if r.TargetAmount.Valid {
		a.TargetAmount = &r.TargetAmount.Float64
	}
	if r.DueDate.Valid {
		a.DueDate = &r.DueDate.String
	}
	if r.Counterparty.Valid {
		a.Counterparty = &r.Counterparty.String
	}
	if r.DebtDescription.Valid {
		a.DebtDesc = &r.DebtDescription.String
	}
	if t, ok := parseNullTime(r.CreatedAt); ok {
		a.CreatedAt = &t
	}
	return a
}

func parseNullTime(raw sql.NullString) (time.Time, bool) {
	if !raw.Valid || raw.String == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw.String, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func nilOr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
