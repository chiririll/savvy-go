package ledger_test

import (
	"testing"
	"time"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// pay writes a transaction with one payment per account (payments are task 2's
// repository; here they are only the rows a balance is read from).
func (f *fixture) pay(t *testing.T, typ, status, date string, minor map[int64]int64) int64 {
	t.Helper()
	res, err := f.repo.DB.ExecContext(
		ctx, `INSERT INTO transactions (type, status, date) VALUES (?, ?, ?)`, typ, status, date,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	for account, amount := range minor {
		if _, err := f.repo.DB.ExecContext(
			ctx, `INSERT INTO payments (transaction_id, account_id, amount) VALUES (?, ?, ?)`, id, account, amount,
		); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

// debt makes a debt account in USD and its debts row.
func (f *fixture) debt(t *testing.T, name, direction string, principal int64) int64 {
	t.Helper()
	unit := money.Unit{ID: f.usd, Decimals: 2}
	id, err := f.repo.InsertAccount(ctx, ledgerrepo.AccountFields{
		Name: name, Type: "debt", CurrencyID: f.usd, IsActive: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.InsertDebt(ctx, id, ledgerrepo.DebtFields{
		Direction: direction, Principal: money.New(principal, unit),
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestDailyChangesAreTheSumOfPayments(t *testing.T) {
	f := newFixture(t)
	f.pay(t, "income", "confirmed", "2026-01-02", map[int64]int64{f.cash: 5000})
	f.pay(t, "expense", "confirmed", "2026-01-02", map[int64]int64{f.cash: -1200})
	f.pay(t, "expense", "confirmed", "2026-01-05", map[int64]int64{f.cash: -300})
	// A transfer moves money on both of its accounts.
	f.pay(t, "transfer", "confirmed", "2026-01-05", map[int64]int64{f.cash: -1000, f.yen: 150000})
	f.pay(t, "expense", "pending", "2026-01-06", map[int64]int64{f.cash: -999})
	f.pay(t, "expense", "skipped", "2026-01-06", map[int64]int64{f.cash: -999})

	got, err := f.repo.DailyChanges(ctx, f.cash, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []ledgerrepo.DayChange{{Day: "2026-01-02", Delta: 3800}, {Day: "2026-01-05", Delta: -1300}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("cash changes = %+v, want %+v", got, want)
	}
	if yen, _ := f.repo.DailyChanges(ctx, f.yen, ""); len(yen) != 1 || yen[0].Delta != 150000 {
		t.Fatalf("yen changes = %+v", yen)
	}
	if early, _ := f.repo.DailyChanges(ctx, f.cash, "2026-01-04"); len(early) != 1 || early[0].Day != "2026-01-02" {
		t.Fatalf("changes as of 2026-01-04 = %+v", early)
	}
}

func TestServiceAccountsStayOutOfPickers(t *testing.T) {
	f := newFixture(t)
	debt := f.debt(t, "Ann", "owed_to_me", 1000)
	space, err := f.repo.InsertAccount(ctx, ledgerrepo.AccountFields{
		Name: "Other spaces · USD", Type: "space", CurrencyID: f.usd, IsActive: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	picker, _ := f.repo.Accounts(ctx, ledgerrepo.AccountFilter{ExcludeService: true})
	if len(picker) != 2 {
		t.Fatalf("a picker lists %d accounts, want the 2 ordinary ones", len(picker))
	}
	all, _ := f.repo.Accounts(ctx, ledgerrepo.AccountFilter{})
	if len(all) != 4 {
		t.Fatalf("all accounts = %d, want 4", len(all))
	}
	for _, a := range all {
		if want := a.ID == debt || a.ID == space; a.IsService != want {
			t.Errorf("%s: IsService = %v, want %v", a.Name, a.IsService, want)
		}
	}
	if last := all[len(all)-1]; last.ID != space {
		t.Errorf("the space account sorts last, got %s", last.Name)
	}
}

func TestSpaceAccountIsFoundByCurrency(t *testing.T) {
	f := newFixture(t)
	if id, err := f.repo.SpaceAccountID(ctx, f.usd); id != nil || err != nil {
		t.Fatalf("before it is made: %v, %v", id, err)
	}
	made, err := f.repo.InsertAccount(ctx, ledgerrepo.AccountFields{
		Name: "Other spaces · USD", Type: "space", CurrencyID: f.usd, IsActive: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := f.repo.SpaceAccountID(ctx, f.usd); id == nil || *id != made {
		t.Fatalf("found %v, want %d", id, made)
	}
	if id, _ := f.repo.SpaceAccountID(ctx, f.jpy); id != nil {
		t.Fatalf("a space account of another currency was found: %d", *id)
	}
}

func TestDebtRoundTrip(t *testing.T) {
	f := newFixture(t)
	id := f.debt(t, "Ann", "i_owe", 5000)

	d, err := f.repo.Debt(ctx, id)
	if err != nil || d == nil {
		t.Fatalf("debt: %v, %v", d, err)
	}
	if d.Direction != "i_owe" || d.Principal.Minor() != 5000 || d.Principal.Unit().Decimals != 2 ||
		d.ClosedAt != nil || d.DueDate != nil || d.Name != "Ann" || !d.IsService {
		t.Fatalf("debt = %+v", d)
	}
	if other, _ := f.repo.Debt(ctx, f.cash); other != nil {
		t.Fatal("an ordinary account is not a debt")
	}

	due, who, note := "2026-06-01", "Ann", "rent"
	if err := f.repo.UpdateDebt(ctx, id, ledgerrepo.DebtFields{
		Direction: "owed_to_me", Principal: money.New(7000, d.Principal.Unit()), DueDate: &due, Counterparty: &who, Note: &note,
	}); err != nil {
		t.Fatal(err)
	}
	d, _ = f.repo.Debt(ctx, id)
	if d.Direction != "owed_to_me" || d.Principal.Minor() != 7000 || *d.DueDate != due || *d.Counterparty != who ||
		*d.Note != note {
		t.Fatalf("updated debt = %+v", d)
	}

	closedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	if err := f.repo.CloseDebt(ctx, id, closedAt); err != nil {
		t.Fatal(err)
	}
	if open, _ := f.repo.Debts(ctx, true); len(open) != 0 {
		t.Fatalf("a closed debt is listed as open: %+v", open)
	}
	all, _ := f.repo.Debts(ctx, false)
	if len(all) != 1 || all[0].ClosedAt == nil || !all[0].ClosedAt.Equal(closedAt) {
		t.Fatalf("all debts = %+v", all)
	}
	if err := f.repo.ReopenDebt(ctx, id); err != nil {
		t.Fatal(err)
	}
	if open, _ := f.repo.Debts(ctx, true); len(open) != 1 || open[0].ClosedAt != nil {
		t.Fatalf("reopened debt = %+v", open)
	}
}

func TestDebtHistoryDoesNotCountTheIssuance(t *testing.T) {
	f := newFixture(t)
	id := f.debt(t, "Ann", "owed_to_me", 5000)
	f.pay(t, "debt_lend", "confirmed", "2026-01-01", map[int64]int64{f.cash: -5000, id: 5000})
	if n := f.repo.CountDebtHistory(ctx, id); n != 0 {
		t.Fatalf("history after the issuance = %d", n)
	}
	f.pay(t, "debt_collection", "confirmed", "2026-01-10", map[int64]int64{f.cash: 1000, id: -1000})
	if n := f.repo.CountDebtHistory(ctx, id); n != 1 {
		t.Fatalf("history after a collection = %d", n)
	}
	if n := f.repo.CountAccountPayments(ctx, f.cash); n != 2 {
		t.Fatalf("payments on cash = %d", n)
	}
}

func TestDeleteDebtRemovesItsIssuance(t *testing.T) {
	f := newFixture(t)
	id := f.debt(t, "Ann", "owed_to_me", 5000)
	lend := f.pay(t, "debt_lend", "confirmed", "2026-01-01", map[int64]int64{f.cash: -5000, id: 5000})
	unrelated := f.pay(t, "expense", "confirmed", "2026-01-02", map[int64]int64{f.cash: -100})

	if err := f.repo.DeleteDebt(ctx, id); err != nil {
		t.Fatal(err)
	}
	if d, _ := f.repo.Debt(ctx, id); d != nil {
		t.Fatal("the debt is still there")
	}
	if a, _ := f.repo.Account(ctx, id); a != nil {
		t.Fatal("the debt account is still there")
	}
	var left int
	row := f.repo.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions WHERE id IN (?, ?)`, lend, unrelated)
	if err := row.Scan(&left); err != nil || left != 1 {
		t.Fatalf("transactions left = %d, %v; want only the unrelated one", left, err)
	}
	if n := f.repo.CountAccountPayments(ctx, f.cash); n != 1 {
		t.Fatalf("payments on cash = %d, want 1", n)
	}
}

func TestChangesAfterADateExcludeThatDay(t *testing.T) {
	f := newFixture(t)
	f.pay(t, "income", "confirmed", "2026-01-02", map[int64]int64{f.cash: 5000})
	f.pay(t, "expense", "confirmed", "2026-01-05", map[int64]int64{f.cash: -300})
	f.pay(t, "expense", "confirmed", "2026-01-09", map[int64]int64{f.cash: -100})
	f.pay(t, "expense", "pending", "2026-01-07", map[int64]int64{f.cash: -999})

	got, err := f.repo.DailyChangesBetween(ctx, f.cash, "2026-01-02", "")
	if err != nil || len(got) != 2 || got[0].Day != "2026-01-05" || got[1].Day != "2026-01-09" {
		t.Fatalf("after 01-02 = %+v %v", got, err)
	}
	if got, _ = f.repo.DailyChangesBetween(ctx, f.cash, "2026-01-02", "2026-01-08"); len(got) != 1 || got[0].Delta != -300 {
		t.Fatalf("(01-02, 01-08] = %+v", got)
	}
	for _, tc := range []struct {
		after, asOf string
		want        int64
	}{
		{"", "", 4600}, {"2026-01-02", "", -400}, {"2026-01-05", "2026-01-09", -100},
		{"2026-01-09", "", 0}, {"2026-01-01", "2026-01-05", 4700},
	} {
		sum, err := f.repo.DeltaSumBetween(ctx, f.cash, tc.after, tc.asOf)
		if err != nil || sum != tc.want {
			t.Errorf("sum (%q, %q] = %d %v, want %d", tc.after, tc.asOf, sum, err, tc.want)
		}
	}
}
