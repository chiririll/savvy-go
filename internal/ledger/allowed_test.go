package ledger

import (
	"errors"
	"testing"

	"savvy-go/internal/money"
)

// The restriction matrix of docs/plans/transactions-erd.md: what may be done to
// a transaction follows from its status and the reasons derived from its data.
func TestTransactionAllowed(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		reasons []Reason
		want    TxActions
	}{
		{"confirmed", "confirmed", nil, TxActions{Edit: true, Delete: true, Duplicate: true, EditDetails: true}},
		{"pending by hand", "pending", nil,
			TxActions{Edit: true, Delete: true, Duplicate: true, Confirm: true, EditDetails: true}},
		{"pending occurrence", "pending", []Reason{ReasonRecurringPending}, TxActions{Confirm: true, Skip: true}},
		{"skipped", "skipped", nil, TxActions{Delete: true}},
		{"space transfer", "confirmed", []Reason{ReasonSpaceTransfer}, TxActions{Edit: true, EditDetails: true}},
		{"space transfer in review", "confirmed", []Reason{ReasonSpaceTransfer, ReasonTransferReview}, TxActions{}},
		{"debt", "confirmed", []Reason{ReasonDebt}, TxActions{EditDetails: true}},
		{"a reason nobody mapped", "confirmed", []Reason{"unknown"}, TxActions{}},
	}
	for _, c := range cases {
		got := Transaction{Type: "expense", Status: c.status, Reasons: c.reasons}.Allowed()
		if got != c.want {
			t.Errorf("%s: Allowed() = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestRefusalNamesTheReason(t *testing.T) {
	tx := Transaction{Status: "confirmed", Reasons: []Reason{ReasonDebt}}
	err := tx.refuse("edit")
	var re *RestrictedError
	if !errors.Is(err, ErrRestricted) || !errors.As(err, &re) || re.Reason != ReasonDebt {
		t.Fatalf("refuse = %v, want a restriction by %q", err, ReasonDebt)
	}
	if tx.refuse("edit_details") != nil {
		t.Error("a debt transaction can be described")
	}
}

var testUnit = money.Unit{ID: 1, Decimals: 2}

// debtOwing is a debt I owe: its balance is minus what is still to be paid.
func debtOwing(principal *int64, owed int64) Debt {
	d := Debt{Account: Account{Balance: money.New(-owed, testUnit)}, Direction: DebtIOwe, Principal: money.Zero(testUnit)}
	if principal != nil {
		d.Principal = money.New(*principal, testUnit)
	}
	return d
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
		if got := debtOwing(c.target, c.balance).PaymentProgress(); got != c.want {
			t.Errorf("%s: PaymentProgress() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOwedFollowsDirection(t *testing.T) {
	owe := Debt{Account: Account{Balance: money.New(-500, testUnit)}, Direction: DebtIOwe}
	owed := Debt{Account: Account{Balance: money.New(500, testUnit)}, Direction: DebtOwedToMe}
	if owe.Owed().Minor() != 500 || owed.Owed().Minor() != 500 {
		t.Fatalf("owed %d and %d, want 500 both", owe.Owed().Minor(), owed.Owed().Minor())
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
