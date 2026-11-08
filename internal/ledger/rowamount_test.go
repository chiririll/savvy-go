package ledger

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

var usd = Currency{ID: 3, Code: "USD", Decimals: 2, Rate: decimal.RequireFromString("0.9")}

func payment(a Account, minor int64) Payment {
	return Payment{AccountID: a.ID, Account: &a, Amount: money.New(minor, a.Currency.Unit())}
}

func lineOf(c Currency, minor int64) Line { return Line{Amount: money.New(minor, c.Unit())} }

func TestRowAmountOneCurrency(t *testing.T) {
	bank := acct(1, "bank", eur)
	tx := Transaction{
		Type:     "expense",
		Payments: []Payment{payment(bank, -5000)},
		Lines:    []Line{lineOf(eur, 3000), lineOf(eur, 2000)},
	}
	got, ok := tx.RowAmount()
	if !ok || got.Converted || got.Main.Amount.Minor() != 5000 || got.Main.Currency.Code != "EUR" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	if len(got.Totals) != 1 {
		t.Fatalf("totals = %+v", got.Totals)
	}
}

func TestRowAmountSeveralCurrenciesConvertsToTheFirst(t *testing.T) {
	bank, cash := acct(1, "bank", eur), acct(2, "cash", usd)
	tx := Transaction{
		Type:     "expense",
		Payments: []Payment{payment(bank, -1000), payment(cash, -1000)},
		Lines:    []Line{lineOf(eur, 1000), lineOf(usd, 1000)},
	}
	got, ok := tx.RowAmount()
	if !ok || !got.Converted {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	// 10.00 EUR + 10.00 USD at 0.9 EUR per USD = 19.00 EUR, in the first currency.
	if got.Main.Currency.Code != "EUR" || got.Main.Amount.Minor() != 1900 {
		t.Fatalf("main = %+v", got.Main)
	}
	if len(got.Totals) != 2 || got.Totals[1].Currency.Code != "USD" || got.Totals[1].Amount.Minor() != 1000 {
		t.Fatalf("totals = %+v", got.Totals)
	}
}

func TestRowAmountWithoutLinesReadsOrdinaryPayments(t *testing.T) {
	bank, debt := acct(1, "bank", eur), acct(2, "debt", eur)
	tx := Transaction{Type: "debt_payment", Payments: []Payment{payment(bank, -700), payment(debt, 700)}}
	got, ok := tx.RowAmount()
	if !ok || got.Main.Amount.Minor() != 700 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestRowAmountNothingToShow(t *testing.T) {
	if _, ok := (Transaction{Type: "expense"}).RowAmount(); ok {
		t.Fatal("a transaction without lines or payments has no amount")
	}
}

func TestTransferShowsSentAndReceived(t *testing.T) {
	from, to := acct(1, "bank", eur), acct(2, "cash", rsd)
	tx := Transaction{
		Type:     "transfer",
		Payments: []Payment{payment(from, -1000), payment(to, 117000)},
		Lines:    []Line{lineOf(eur, 50)}, // the fee
	}
	sent, received := tx.ListAmounts()
	if sent == nil || sent.Main.Currency.Code != "EUR" || sent.Main.Amount.Minor() != 1000 {
		t.Fatalf("sent = %+v", sent)
	}
	if received == nil || received.Main.Currency.Code != "RSD" || received.Main.Amount.Minor() != 117000 {
		t.Fatalf("received = %+v", received)
	}
}

func TestListAmountsOfAnExpenseHasNoReceived(t *testing.T) {
	bank := acct(1, "bank", eur)
	tx := Transaction{Type: "expense", Payments: []Payment{payment(bank, -100)}, Lines: []Line{lineOf(eur, 100)}}
	main, received := tx.ListAmounts()
	if main == nil || received != nil {
		t.Fatalf("main=%+v received=%+v", main, received)
	}
}

func dayTx(date, status, typ string, a Account, minor int64) Transaction {
	d := date
	return Transaction{
		Date: &d, Status: status, Type: typ,
		Payments: []Payment{payment(a, minor)},
		Lines:    []Line{lineOf(*a.Currency, abs64(minor))},
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestDayTotalsNetPerCurrencyAndSkipSkipped(t *testing.T) {
	bank, cash := acct(1, "bank", eur), acct(2, "cash", usd)
	txs := []Transaction{
		dayTx("2026-11-01", "confirmed", "income", bank, 10000),
		dayTx("2026-11-01", "confirmed", "expense", bank, -2500),
		dayTx("2026-11-01", "confirmed", "expense", cash, -100),
		dayTx("2026-11-01", "skipped", "expense", bank, -99999),
		dayTx("2026-11-02", "pending", "expense", bank, -300),
	}
	days := DayTotals(txs)
	if len(days) != 2 || days[0].Date != "2026-11-01" || days[1].Date != "2026-11-02" {
		t.Fatalf("days = %+v", days)
	}
	first := days[0].Totals
	if len(first) != 2 {
		t.Fatalf("totals = %+v", first)
	}
	// Largest by value in the base currency first: 75.00 EUR before 1.00 USD (0.90 EUR).
	if first[0].Currency.Code != "EUR" || first[0].Amount.Minor() != 7500 {
		t.Fatalf("first = %+v", first[0])
	}
	if first[1].Currency.Code != "USD" || first[1].Amount.Minor() != -100 {
		t.Fatalf("second = %+v", first[1])
	}
}

func TestDayTotalsTransferCountsOnlyItsFee(t *testing.T) {
	from, to := acct(1, "bank", eur), acct(2, "cash", rsd)
	d := "2026-11-01"
	tx := Transaction{
		Date: &d, Status: "confirmed", Type: "transfer",
		Payments: []Payment{payment(from, -1050), payment(to, 117000)},
		Lines:    []Line{lineOf(eur, 50)},
	}
	days := DayTotals([]Transaction{tx})
	if len(days) != 1 || len(days[0].Totals) != 1 || days[0].Totals[0].Amount.Minor() != -50 {
		t.Fatalf("days = %+v", days)
	}
}
