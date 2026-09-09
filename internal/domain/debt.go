package domain

import (
	"context"
	"fmt"
	"time"
)

func (a Account) DebtJSON() map[string]any {
	target := 0.0
	if a.TargetAmount != nil {
		target = *a.TargetAmount
	}
	remaining := a.Balance
	progress := 0.0
	if target > 0 {
		progress = (target - remaining) / target * 100
		if progress < 0 {
			progress = 0
		}
	}
	label := ""
	if a.DebtType != nil {
		switch *a.DebtType {
		case "i_owe":
			label = "I owe"
		case "owed_to_me":
			label = "Owed to me"
		}
	}
	m := a.JSON()
	m["debtType"] = a.DebtType
	m["debtTypeLabel"] = label
	m["targetAmount"] = target
	m["remainingDebt"] = remaining
	m["paymentProgress"] = progress
	m["dueDate"] = a.DueDate
	m["counterparty"] = a.Counterparty
	m["description"] = a.DebtDesc
	m["isPaidOff"] = a.IsPaidOff
	return m
}

type Debts struct {
	Accounts     Accounts
	Transactions Transactions
}

func (s Debts) All(ctx context.Context, includeCompleted bool) ([]Account, error) {
	q := accountSelect + ` WHERE a.type = 'debt'`
	if !includeCompleted {
		q += ` AND a.is_paid_off = 0`
	}
	q += ` ORDER BY a.sort_order, a.id`
	return s.Accounts.list(ctx, q)
}

func (s Debts) Create(ctx context.Context, name, debtType string, currencyID, accountID int64, amount float64, date, origin string, due, counter, desc *string) (*Account, error) {
	if origin == "new" && accountID != 0 {
		src, err := s.Accounts.ByID(ctx, accountID)
		if err != nil || src == nil {
			return nil, fmt.Errorf("source account")
		}
		if src.Type == "debt" {
			return nil, fmt.Errorf("cannot use debt source")
		}
		currencyID = src.CurrencyID
	}
	debt, err := s.Accounts.Create(ctx, Account{
		Name: name, Type: "debt", CurrencyID: currencyID,
		InitialBalance: 0, TargetAmount: &amount, DueDate: due,
		Counterparty: counter, DebtDesc: desc, IsActive: true, DebtType: &debtType,
	})
	if err != nil {
		return nil, err
	}
	if origin == "new" && accountID != 0 {
		if err := s.recordIssuance(ctx, *debt, accountID, debtType, amount, date, desc); err != nil {
			return nil, err
		}
		debt, err = s.Accounts.ByID(ctx, debt.ID)
		if err != nil {
			return nil, err
		}
	}
	return debt, nil
}

func (s Debts) recordIssuance(ctx context.Context, debt Account, accountID int64, debtType string, amount float64, date string, desc *string) error {
	typ := "debt_lend"
	amt := amount
	toAmt := amount
	if debtType == "i_owe" {
		typ = "debt_borrow"
	}
	_, err := s.Transactions.Create(ctx, TxInput{
		Type: typ, AccountID: accountID, ToAccountID: &debt.ID,
		Amount: amt, ToAmount: &toAmt, Description: desc, Date: &date,
	})
	return err
}

func (s Debts) Payment(ctx context.Context, debtID, accountID int64, amount float64, date string, desc *string, collect bool) (*Transaction, error) {
	debt, err := s.Accounts.ByID(ctx, debtID)
	if err != nil || debt == nil || debt.Type != "debt" {
		return nil, fmt.Errorf("not a debt")
	}
	if debt.IsPaidOff {
		return nil, fmt.Errorf("already paid off")
	}
	typ := "debt_payment"
	if collect {
		typ = "debt_collection"
	}
	tx, err := s.Transactions.Create(ctx, TxInput{
		Type: typ, AccountID: accountID, ToAccountID: &debt.ID,
		Amount: amount, ToAmount: &amount, Description: desc, Date: &date,
	})
	if err != nil {
		return nil, err
	}
	s.maybePayOff(ctx, debt)
	return tx, nil
}

func (s Debts) maybePayOff(ctx context.Context, debt *Account) {
	fresh, err := s.Accounts.ByID(ctx, debt.ID)
	if err != nil || fresh == nil {
		return
	}
	if fresh.Balance <= 0.0001 {
		_, _ = s.Accounts.DB.ExecContext(ctx, `UPDATE accounts SET is_paid_off = 1 WHERE id = ?`, debt.ID)
	}
}

func (s Debts) Reopen(ctx context.Context, id int64) (*Account, error) {
	d, err := s.Accounts.ByID(ctx, id)
	if err != nil || d == nil || d.Type != "debt" {
		return nil, fmt.Errorf("not a debt")
	}
	if !d.IsPaidOff {
		return nil, fmt.Errorf("not paid off")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.Accounts.DB.ExecContext(ctx, `UPDATE accounts SET is_paid_off = 0, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return nil, err
	}
	return s.Accounts.ByID(ctx, id)
}

func (s Debts) Summary(ctx context.Context) map[string]any {
	debts, _ := s.All(ctx, false)
	var iOwe, owed float64
	for _, d := range debts {
		amt := d.Balance
		if d.Currency != nil && !d.Currency.IsBase {
			amt = d.Currency.ConvertToBase(amt)
		}
		if d.DebtType != nil && *d.DebtType == "i_owe" {
			iOwe += amt
		} else {
			owed += amt
		}
	}
	return map[string]any{
		"total_i_owe": iOwe, "total_owed_to_me": owed, "net_debt": owed - iOwe,
		"debts_count": len(debts), "currency": nil, "decimals": 2,
	}
}

func (s Debts) Delete(ctx context.Context, id int64) error {
	var n int
	_ = s.Accounts.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM transactions WHERE to_account_id = ? AND type IN ('debt_payment','debt_collection')`, id).Scan(&n)
	if n > 0 {
		return fmt.Errorf("has history")
	}
	_, _ = s.Accounts.DB.ExecContext(ctx, `DELETE FROM transactions WHERE to_account_id = ? AND type IN ('debt_lend','debt_borrow')`, id)
	_, err := s.Accounts.DB.ExecContext(ctx, `DELETE FROM accounts WHERE id = ? AND type = 'debt'`, id)
	return err
}
