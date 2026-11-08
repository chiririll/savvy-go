package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"savvy-go/internal/db/filter"
)

func (e *moneyEnv) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := e.db.ExecContext(e.ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

// Paying an expense from two accounts in two currencies, with change back, is
// one transaction: payments are what moved, lines what it was for.
func TestExpenseWithSeveralPaymentsAndChange(t *testing.T) {
	e := newMoneyEnv(t)
	usd, jpy := dec("-20"), dec("1000") // 1000 JPY back is 10 USD at 0.01
	date := "2024-02-01"
	tx, err := e.txs.Create(e.ctx, TxInput{
		Type: "expense", Date: &date,
		Payments: []PaymentInput{{AccountID: e.usdAcc.ID, Amount: &usd}, {AccountID: e.jpyAcc.ID, Amount: &jpy}},
		Lines:    []LineInput{{CurrencyID: e.usd.ID, Amount: dec("10"), Note: ptr("bus")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Difference != nil {
		t.Errorf("difference = %s, want none: 20 USD paid, 10 USD back, 10 USD of lines", tx.Difference)
	}
	if len(tx.Payments) != 2 || len(tx.Lines) != 1 || tx.Lines[0].Amount.Unit() != e.usd.Unit() {
		t.Fatalf("payments %+v lines %+v", tx.Payments, tx.Lines)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("80")) {
		t.Errorf("usd = %s, want 80", got)
	}
	if got := e.balance(t, e.jpyAcc.ID); !got.Equal(dec("4000")) {
		t.Errorf("jpy = %s, want 4000", got)
	}
}

// The soft check reports; it never blocks the save.
func TestCreateReportsADifferenceAndStillSaves(t *testing.T) {
	e := newMoneyEnv(t)
	x := simple("expense", e.usdAcc.ID, "15")
	x.Lines[0].Amount = dec("10")
	date := "2024-02-01"
	x.Date = &date
	tx, err := e.txs.Create(e.ctx, x)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Difference == nil || tx.Difference.Decimal().String() != "5" {
		t.Fatalf("difference = %v, want 5 USD", tx.Difference)
	}
	again, _ := e.txs.ByID(e.ctx, tx.ID)
	if again.Difference != nil {
		t.Error("a transaction read back carries no difference: only a save computes it")
	}
}

func TestUpdateRewritesLinesKeepsTheRest(t *testing.T) {
	e := newMoneyEnv(t)
	food, err := (Categories{DB: e.db}).Create(e.ctx, Category{Name: "Food", Type: "expense"})
	if err != nil {
		t.Fatal(err)
	}
	tx := e.tx(t, "expense", e.usdAcc.ID, "10")
	next, err := e.txs.Update(e.ctx, tx.ID, TxInput{
		Lines: []LineInput{
			{CategoryID: &food.ID, Amount: dec("6"), Note: ptr("a")},
			{CategoryID: &food.ID, Amount: dec("4"), Note: ptr("b")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Lines) != 2 || next.Lines[1].Seq != 1 || len(next.Payments) != 1 ||
		!next.Payments[0].Amount.Decimal().Equal(dec("-10")) || next.Date == nil || next.Type != "expense" {
		t.Fatalf("after update: payments %+v lines %+v date %v", next.Payments, next.Lines, next.Date)
	}
}

// Debt transactions belong to the debts module: only their details change.
func TestDebtTransactionOnlyAllowsDetails(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	debt, err := debts.Create(e.ctx, DebtInput{Name: "loan", Direction: DebtIOwe, CurrencyID: e.usd.ID, Amount: dec("100")})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := debts.Payment(e.ctx, debt.ID, e.usdAcc.ID, dec("40"), "2024-01-15", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tx.Reasons, []Reason{ReasonDebt}) {
		t.Fatalf("reasons = %v, want debt", tx.Reasons)
	}
	if got := tx.Allowed(); got != (TxActions{EditDetails: true}) {
		t.Fatalf("allowed = %+v, want details only", got)
	}
	changed := dec("-999")
	moved := tx.Input()
	moved.Payments[0].Amount = &changed
	if _, err := e.txs.Update(e.ctx, tx.ID, moved); !errors.Is(err, ErrDetailsOnly) {
		t.Errorf("changing the money: %v, want ErrDetailsOnly", err)
	}
	described := tx.Input()
	described.Description = ptr("March instalment")
	got, err := e.txs.Update(e.ctx, tx.ID, described)
	if err != nil || got.Description == nil || *got.Description != "March instalment" {
		t.Errorf("describing it: %v %+v", err, got)
	}
	if err := e.txs.Delete(e.ctx, tx.ID); !errors.Is(err, ErrRestricted) {
		t.Errorf("delete = %v, want a restriction", err)
	}
	if _, err := e.txs.Duplicate(e.ctx, tx.ID); !errors.Is(err, ErrRestricted) {
		t.Errorf("duplicate = %v, want a restriction", err)
	}
	if err := e.txs.As(ReasonDebt).Delete(e.ctx, tx.ID); err != nil {
		t.Errorf("the debts module deletes its own: %v", err)
	}
}

// A transaction cannot slip money onto a service account around its module.
func TestServiceAccountsAreOwnedByTheirModule(t *testing.T) {
	e := newMoneyEnv(t)
	debt, err := newDebts(e).Create(e.ctx, DebtInput{Name: "loan", Direction: DebtIOwe, CurrencyID: e.usd.ID, Amount: dec("10")})
	if err != nil {
		t.Fatal(err)
	}
	out, in, date := dec("-5"), dec("5"), "2024-01-15"
	_, err = e.txs.Create(e.ctx, TxInput{
		Type: "transfer", Date: &date,
		Payments: []PaymentInput{{AccountID: e.usdAcc.ID, Amount: &out}, {AccountID: debt.ID, Amount: &in}},
	})
	if !errors.Is(err, ErrServiceAccount) {
		t.Fatalf("a plain transfer onto a debt: %v, want ErrServiceAccount", err)
	}
}

func spaceTransfer(t *testing.T, e *moneyEnv) *Transaction {
	t.Helper()
	spaceAcct, err := (Accounts{DB: e.db}).SpaceAccount(e.ctx, e.usd.ID)
	if err != nil {
		t.Fatal(err)
	}
	e.exec(t, `INSERT INTO space_transfers (uuid, payload, source_kid)
		VALUES ('u1', '{"from":{"space":"a"},"to":{"space":"b"},"version":1}', 'k')`)
	out, in, date, uuid := dec("-10"), dec("10"), "2024-03-01", "u1"
	tx, err := e.txs.As(ReasonSpaceTransfer).Create(e.ctx, TxInput{
		Type: "transfer_out", Date: &date,
		Payments: []PaymentInput{
			{AccountID: e.usdAcc.ID, Amount: &out},
			{AccountID: spaceAcct, Amount: &in, SpaceTransferUUID: &uuid},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

// One side of a transfer between spaces: details change, the transfer's own
// payments and date only through internal/transfers, and review freezes it.
func TestSpaceTransferSide(t *testing.T) {
	e := newMoneyEnv(t)
	tx := spaceTransfer(t, e)
	if !slices.Equal(tx.Reasons, []Reason{ReasonSpaceTransfer}) {
		t.Fatalf("reasons = %v", tx.Reasons)
	}
	described := tx.Input()
	described.Description = ptr("rent share")
	if _, err := e.txs.Update(e.ctx, tx.ID, described); err != nil {
		t.Errorf("describing a side: %v", err)
	}
	moved := tx.Input()
	other := dec("-11")
	moved.Payments[0].Amount = &other
	if _, err := e.txs.Update(e.ctx, tx.ID, moved); !errors.Is(err, ErrRestricted) {
		t.Errorf("changing the paired payment: %v, want a restriction", err)
	}
	if err := e.txs.Delete(e.ctx, tx.ID); !errors.Is(err, ErrRestricted) {
		t.Errorf("delete = %v, want a restriction", err)
	}
	// Saving keeps the uuid on its payment.
	again, err := e.txs.Update(e.ctx, tx.ID, described)
	if err != nil {
		t.Fatal(err)
	}
	var kept int
	for _, p := range again.Payments {
		if p.SpaceTransferUUID != nil && *p.SpaceTransferUUID == "u1" {
			kept++
		}
	}
	if kept != 1 {
		t.Errorf("uuid on %d payments after saving, want 1", kept)
	}
	e.exec(t, `UPDATE space_transfers SET status = 'in_review', remote = payload`)
	review, _ := e.txs.ByID(e.ctx, tx.ID)
	if !slices.Contains(review.Reasons, ReasonTransferReview) || review.Allowed() != (TxActions{}) {
		t.Errorf("in review: reasons %v allowed %+v, want nothing", review.Reasons, review.Allowed())
	}
	if _, err := e.txs.Update(e.ctx, tx.ID, described); !errors.Is(err, ErrRestricted) {
		t.Errorf("update in review = %v, want a restriction", err)
	}
}

// occurrence makes the pending transaction of a template for a new rule.
func (e *moneyEnv) occurrence(t *testing.T, template string) (ruleID, txID int64) {
	t.Helper()
	e.exec(t, `INSERT INTO recurring_transactions (template, schedule) VALUES (?, '{}')`, template)
	if err := e.db.QueryRowContext(e.ctx, `SELECT MAX(id) FROM recurring_transactions`).Scan(&ruleID); err != nil {
		t.Fatal(err)
	}
	id, err := (Templates{DB: e.db}).Create(e.ctx, json.RawMessage(template), "2024-04-01")
	if err != nil {
		t.Fatal(err)
	}
	e.exec(t, `INSERT INTO recurring_instances (recurring_transaction_id, due_date, transaction_id)
		VALUES (?, '2024-04-01', ?)`, ruleID, id)
	return ruleID, id
}

// transferTemplate moves 15 of the first account; the second takes what is left.
const transferTemplate = `{"type":"transfer","payments":[{"account_id":%d,"amount":"-15"},` +
	`{"account_id":%d,"amount":null,"estimated":true}]}`

// A payment without an amount takes what balances the others at the rates of
// the day: 15 USD out is 1500 JPY in.
func TestTemplateBalancesItsEmptyPayment(t *testing.T) {
	e := newMoneyEnv(t)
	_, id := e.occurrence(t, fmt.Sprintf(transferTemplate, e.usdAcc.ID, e.jpyAcc.ID))
	tx, _ := e.txs.ByID(e.ctx, id)
	if tx.Status != "pending" || tx.Date == nil || *tx.Date != "2024-04-01" || len(tx.Payments) != 2 {
		t.Fatalf("occurrence = %+v", tx)
	}
	for _, p := range tx.Payments {
		want := map[int64]string{e.usdAcc.ID: "-15", e.jpyAcc.ID: "1500"}[p.AccountID]
		if p.Amount.Decimal().String() != want {
			t.Errorf("account %d got %s, want %s", p.AccountID, p.Amount, want)
		}
	}
}

func TestTemplateValidation(t *testing.T) {
	e := newMoneyEnv(t)
	tpl := Templates{DB: e.db}
	usd, jpy := e.usdAcc.ID, e.jpyAcc.ID
	cases := map[string]struct {
		json string
		want error
	}{
		"fine":        {fmt.Sprintf(transferTemplate, usd, jpy), nil},
		"unknown key": {`{"type":"expense","amout":"1"}`, ErrTemplate},
		"unknown account": {
			`{"type":"expense","payments":[{"account_id":999,"amount":"-1"}],"lines":[{"amount":"1"}]}`, ErrUnknownAccount,
		},
		"two empty payments": {
			fmt.Sprintf(`{"type":"transfer","payments":[{"account_id":%d},{"account_id":%d}]}`, usd, jpy), ErrManyBalancing,
		},
		"a debt type": {
			fmt.Sprintf(`{"type":"debt_lend","payments":[{"account_id":%d,"amount":"-1"}]}`, usd), ErrTxType,
		},
	}
	for name, c := range cases {
		if err := tpl.Validate(e.ctx, json.RawMessage(c.json)); !errors.Is(err, c.want) {
			t.Errorf("%s: Validate = %v, want %v", name, err, c.want)
		}
	}
}

// The open occurrence of a rule is confirmed or skipped, not edited or
// deleted; the rule replans it.
func TestRecurringOccurrenceRestrictions(t *testing.T) {
	e := newMoneyEnv(t)
	_, id := e.occurrence(t, fmt.Sprintf(transferTemplate, e.usdAcc.ID, e.jpyAcc.ID))
	tx, _ := e.txs.ByID(e.ctx, id)
	if !slices.Equal(tx.Reasons, []Reason{ReasonRecurringPending}) || tx.Allowed() != (TxActions{Confirm: true, Skip: true}) {
		t.Fatalf("reasons %v allowed %+v", tx.Reasons, tx.Allowed())
	}
	if _, err := e.txs.Update(e.ctx, id, tx.Input()); !errors.Is(err, ErrRestricted) {
		t.Errorf("update = %v, want a restriction", err)
	}
	if err := e.txs.Delete(e.ctx, id); !errors.Is(err, ErrRestricted) {
		t.Errorf("delete = %v, want a restriction", err)
	}

	// The rule rewrites it: 30 USD out is 3000 JPY in, on a later due date.
	changed := fmt.Sprintf(`{"type":"transfer","payments":[{"account_id":%d,"amount":"-30"},{"account_id":%d}]}`,
		e.usdAcc.ID, e.jpyAcc.ID)
	if err := (Templates{DB: e.db}).Replan(e.ctx, id, json.RawMessage(changed), "2024-04-05"); err != nil {
		t.Fatal(err)
	}
	tx, _ = e.txs.ByID(e.ctx, id)
	for _, p := range tx.Payments {
		if p.AccountID == e.jpyAcc.ID && p.Amount.Decimal().String() != "3000" {
			t.Errorf("replanned yen payment = %s, want 3000", p.Amount)
		}
	}
	if *tx.Date != "2024-04-05" {
		t.Errorf("replanned date = %s", *tx.Date)
	}

	// Confirming can settle the real amounts, which an ordinary pending one cannot.
	usd, jpy := dec("-30"), dec("2900")
	done, err := e.txs.Confirm(e.ctx, id, "2024-04-05", &Amendment{Payments: []PaymentInput{
		{AccountID: e.usdAcc.ID, Amount: &usd}, {AccountID: e.jpyAcc.ID, Amount: &jpy},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "confirmed" || done.RecurringID == nil {
		t.Fatalf("confirmed = %+v", done)
	}
	if got := e.balance(t, e.jpyAcc.ID); !got.Equal(dec("5900")) {
		t.Errorf("jpy = %s, want 5900 after the real amount", got)
	}
	if !done.Allowed().Edit {
		t.Error("once confirmed it is an ordinary transaction")
	}
	plain := e.tx(t, "expense", e.usdAcc.ID, "1")
	if _, err := e.txs.Confirm(e.ctx, plain.ID, "", nil); err == nil {
		t.Error("confirming a confirmed transaction must fail")
	}
}

func TestOnlyAnOccurrenceCanBeSkipped(t *testing.T) {
	e := newMoneyEnv(t)
	pending := "pending"
	x := simple("expense", e.usdAcc.ID, "5")
	x.Status = &pending
	by, err := e.txs.Create(e.ctx, x)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.txs.Skip(e.ctx, by.ID); !errors.Is(err, ErrRestricted) {
		t.Errorf("skip a pending one entered by hand = %v, want a restriction", err)
	}
	if _, err := e.txs.Confirm(e.ctx, by.ID, "2024-01-01", nil); err != nil {
		t.Errorf("confirm by hand: %v", err)
	}
}

// A day's total covers every transaction of the day, not those of one page.
func TestDayTotalsCoverTheWholeDayAcrossPages(t *testing.T) {
	e := newMoneyEnv(t)
	for _, day := range []string{"2024-02-01", "2024-02-01", "2024-02-02"} {
		x := simple("expense", e.usdAcc.ID, "10")
		x.Date = ptr(day)
		if _, err := e.txs.Create(e.ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	days, err := e.txs.DayTotals(e.ctx, filter.TxFilter{Dates: []string{"2024-02-01", "2024-02-02"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 2 || days[0].Date != "2024-02-02" || days[1].Date != "2024-02-01" {
		t.Fatalf("days = %+v", days)
	}
	if got := days[1].Totals[0].Amount.Decimal().String(); got != "-20" {
		t.Errorf("2024-02-01 = %s, want -20", got)
	}
	only, _ := e.txs.DayTotals(e.ctx, filter.TxFilter{Dates: []string{"2024-02-02"}})
	if len(only) != 1 {
		t.Errorf("a day outside the filter is counted: %+v", only)
	}
}

// A rule is shown as the transaction it will make, without making one: the open
// payment is worked out at today's rates, nothing is written.
func TestTemplatePreviewIsWhatTheOccurrenceWillBe(t *testing.T) {
	e := newMoneyEnv(t)
	tpl := json.RawMessage(fmt.Sprintf(transferTemplate, e.usdAcc.ID, e.jpyAcc.ID))
	tx, err := (Templates{DB: e.db}).Preview(e.ctx, tpl)
	if err != nil {
		t.Fatal(err)
	}
	sent, received := tx.ListAmounts()
	if sent == nil || sent.Main.Currency.Code != "USD" || sent.Main.Amount.Decimal().String() != "15" {
		t.Fatalf("sent = %+v", sent)
	}
	if received == nil || received.Main.Currency.Code != "JPY" || received.Main.Amount.Decimal().String() != "1500" {
		t.Fatalf("received = %+v", received)
	}
	var n int
	e.db.QueryRowContext(e.ctx, `SELECT COUNT(*) FROM transactions`).Scan(&n)
	if n != 0 {
		t.Errorf("a preview wrote %d transactions", n)
	}
}

func TestTemplatePreviewRefusesABrokenTemplate(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := (Templates{DB: e.db}).Preview(e.ctx, json.RawMessage(`{"type":"expense","amout":"1"}`)); err == nil {
		t.Fatal("a template with an unknown key must not preview")
	}
}
