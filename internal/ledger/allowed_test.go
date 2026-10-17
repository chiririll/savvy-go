package ledger

import (
	"testing"

	"savvy-go/internal/money"
)

func TestTransactionAllowed(t *testing.T) {
	id := int64(7)
	cases := []struct {
		name string
		tx   Transaction
		want TxActions
	}{
		{
			"confirmed",
			Transaction{Type: "expense", Status: "confirmed"},
			TxActions{Edit: true, Delete: true, Duplicate: true},
		},
		{
			"pending",
			Transaction{Type: "expense", Status: "pending"},
			TxActions{Edit: true, Delete: true, Duplicate: true, Confirm: true},
		},
		{
			"pending recurring",
			Transaction{Type: "expense", Status: "pending", RecurringID: &id},
			TxActions{Confirm: true, Skip: true},
		},
		{"confirmed recurring", Transaction{Type: "expense", Status: "confirmed", RecurringID: &id}, TxActions{}},
		{"skipped", Transaction{Type: "expense", Status: "skipped"}, TxActions{Delete: true}},
		{"transfer out", Transaction{Type: "transfer_out", Status: "confirmed"}, TxActions{}},
		{"transfer in", Transaction{Type: "transfer_in", Status: "pending"}, TxActions{}},
	}
	for _, c := range cases {
		if got := c.tx.Allowed(); got != c.want {
			t.Errorf("%s: Allowed() = %+v, want %+v", c.name, got, c.want)
		}
	}
}

var testUnit = money.Unit{ID: 1, Decimals: 2}

func debtAccount(target *int64, balance int64) Account {
	a := Account{Balance: money.New(balance, testUnit)}
	if target != nil {
		m := money.New(*target, testUnit)
		a.TargetAmount = &m
	}
	return a
}

func TestPaymentProgress(t *testing.T) {
	ten := int64(10000)
	cases := []struct {
		name    string
		target  *int64
		balance int64
		want    float64
	}{
		{"nothing paid", &ten, 10000, 0},
		{"a third paid", &ten, 6667, 33.3},
		{"half paid", &ten, 5000, 50},
		{"paid off", &ten, 0, 100},
		{"overpaid is capped", &ten, -2500, 100},
		{"grown beyond target is floored", &ten, 12000, 0},
		{"no target", nil, 5000, 0},
	}
	for _, c := range cases {
		if got := debtAccount(c.target, c.balance).PaymentProgress(); got != c.want {
			t.Errorf("%s: PaymentProgress() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDebtTargetWithoutCurrencyDoesNotPanic(t *testing.T) {
	a := Account{Balance: money.New(500, testUnit)} // Currency is nil
	if got := a.DebtTarget(); !got.IsZero() || got.Unit() != testUnit {
		t.Fatalf("DebtTarget() = %+v", got)
	}
}

func TestSummaryDerivedTotals(t *testing.T) {
	d := DebtSummary{IOwe: money.New(3000, testUnit), OwedToMe: money.New(1000, testUnit)}
	if got := d.Net(); got.Minor() != -2000 {
		t.Errorf("Net() = %d, want -2000", got.Minor())
	}
	s := TransactionSummary{Income: money.New(5000, testUnit), Expense: money.New(1250, testUnit)}
	if got := s.Balance(); got.Minor() != 3750 {
		t.Errorf("Balance() = %d, want 3750", got.Minor())
	}
}
