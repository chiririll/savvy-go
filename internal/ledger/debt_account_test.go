package ledger

import (
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

// addPayments writes a transaction with one payment per account, as minor units
// in each account's currency. Recording transactions is the transactions
// service's job; these tests only need the rows a balance is read from.
func addPayments(t *testing.T, e *moneyEnv, typ, date string, minor map[int64]int64) {
	t.Helper()
	res, err := e.db.ExecContext(e.ctx, `INSERT INTO transactions (type, status, date) VALUES (?, 'confirmed', ?)`, typ, date)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	for account, amount := range minor {
		if _, err := e.db.ExecContext(
			e.ctx, `INSERT INTO payments (transaction_id, account_id, amount) VALUES (?, ?, ?)`, id, account, amount,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func openDebt(t *testing.T, e *moneyEnv, direction, amount string) *Debt {
	t.Helper()
	d, err := newDebts(e).Create(e.ctx, DebtInput{
		Name: direction, Direction: direction, CurrencyID: e.usd.ID, Amount: decimal.RequireFromString(amount),
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDebtBalanceIsSignedByDirection(t *testing.T) {
	e := newMoneyEnv(t)
	owe := openDebt(t, e, DebtIOwe, "50")
	owed := openDebt(t, e, DebtOwedToMe, "30")

	if owe.Balance.Decimal().String() != "-50" || owe.Owed().Decimal().String() != "50" {
		t.Errorf("I owe 50: balance %s, owed %s", owe.Balance, owe.Owed())
	}
	if owed.Balance.Decimal().String() != "30" || owed.Owed().Decimal().String() != "30" {
		t.Errorf("owed to me 30: balance %s, owed %s", owed.Balance, owed.Owed())
	}
	if !owe.IsService || owe.Type != "debt" || owe.Principal.Decimal().String() != "50" {
		t.Errorf("debt account %+v", owe)
	}
	if usd, _ := (Accounts{DB: e.db}).ByID(e.ctx, e.usdAcc.ID); usd.IsService {
		t.Errorf("an ordinary account is not a service account: %+v", usd)
	}
}

func TestPaymentsReduceWhatIsOwed(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debts := newDebts(e)
	owe := openDebt(t, e, DebtIOwe, "50")
	// Paying a debt I owe adds to its (negative) balance; the same for money I
	// collect, which takes from what is owed to me.
	addPayments(t, e, "debt_payment", "2024-01-10", map[int64]int64{e.usdAcc.ID: -2000, owe.ID: 2000})
	got, _ := debts.ByID(e.ctx, owe.ID)
	if got.Owed().Decimal().String() != "30" || got.PaymentProgress() != 40 {
		t.Errorf("after paying 20 of 50: owed %s, progress %v", got.Owed(), got.PaymentProgress())
	}

	owed := openDebt(t, e, DebtOwedToMe, "30")
	addPayments(t, e, "debt_collection", "2024-01-12", map[int64]int64{e.usdAcc.ID: 1000, owed.ID: -1000})
	got, _ = debts.ByID(e.ctx, owed.ID)
	if got.Owed().Decimal().String() != "20" {
		t.Errorf("after collecting 10 of 30: owed %s", got.Owed())
	}

	// The balance of a debt has a history like any account's.
	series, err := accts.BalanceSeries(e.ctx, owe.Account, []string{"2024-01-09", "2024-01-10", "2024-01-31"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-50", "-30", "-30"}
	for i, w := range want {
		if series[i].Decimal().String() != w {
			t.Errorf("series[%d] = %s, want %s", i, series[i], w)
		}
	}
}

func TestBalanceOfAnyAccountIsOpeningPlusPayments(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	addPayments(t, e, "transfer", "2024-01-10", map[int64]int64{e.usdAcc.ID: -2500, e.jpyAcc.ID: 4000})
	usd, _ := accts.ByID(e.ctx, e.usdAcc.ID)
	jpy, _ := accts.ByID(e.ctx, e.jpyAcc.ID)
	if usd.Balance.Decimal().String() != "75" || jpy.Balance.Decimal().String() != "7000" {
		t.Errorf("usd %s, jpy %s; want 75 and 7000", usd.Balance, jpy.Balance)
	}
	if before, _ := accts.Balance(e.ctx, *usd, "2024-01-09"); before.Decimal().String() != "100" {
		t.Errorf("usd the day before = %s", before)
	}
}

func TestListsLeaveDebtsOutForPickers(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debt := openDebt(t, e, DebtIOwe, "10")
	if _, err := accts.SpaceAccount(e.ctx, e.usd.ID); err != nil {
		t.Fatal(err)
	}

	picker, _ := accts.All(e.ctx, true, true)
	if len(picker) != 3 {
		t.Errorf("picker lists %d accounts, want the 3 ordinary ones", len(picker))
	}
	all, _ := accts.All(e.ctx, true, false)
	if len(all) != 5 {
		t.Fatalf("all lists %d accounts, want 5", len(all))
	}
	if list, _ := newDebts(e).All(e.ctx, false); len(list) != 1 || list[0].ID != debt.ID || list[0].Owed().Decimal().String() != "10" {
		t.Errorf("debts = %+v", list)
	}
}

func TestSpaceAccountIsMadeOncePerCurrency(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	first, err := accts.SpaceAccount(e.ctx, e.usd.ID)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := accts.SpaceAccount(e.ctx, e.usd.ID)
	other, _ := accts.SpaceAccount(e.ctx, e.jpy.ID)
	if again != first || other == first {
		t.Errorf("ids %d, %d, %d: want one per currency", first, again, other)
	}
	got, _ := accts.ByID(e.ctx, first)
	if got.Type != "space" || !got.IsService || got.CurrencyID != e.usd.ID || got.Name != "Other spaces · USD" {
		t.Errorf("space account %+v", got)
	}
	if _, err := accts.SpaceAccount(e.ctx, 9999); err != ErrUnknownCurrency {
		t.Errorf("unknown currency: %v", err)
	}
}

func TestDebtNeedsAnAmountAndADirection(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	if _, err := debts.Create(e.ctx, DebtInput{Name: "a", Direction: DebtIOwe, CurrencyID: e.usd.ID}); err != ErrDebtPrincipal {
		t.Errorf("no amount: %v", err)
	}
	if _, err := debts.Create(e.ctx, DebtInput{
		Name: "b", Direction: DebtIOwe, CurrencyID: e.usd.ID, Amount: decimal.NewFromInt(-5),
	}); err != ErrDebtPrincipal {
		t.Errorf("negative amount: %v", err)
	}
	if _, err := debts.Create(e.ctx, DebtInput{
		Name: "c", Direction: "gift", CurrencyID: e.usd.ID, Amount: decimal.NewFromInt(5),
	}); err != ErrDebtDirection {
		t.Errorf("unknown direction: %v", err)
	}
	if list, _ := debts.All(e.ctx, true); len(list) != 0 {
		t.Errorf("a refused debt left %d accounts behind", len(list))
	}
}

func TestChangingThePrincipalKeepsWhatWasPaid(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	debt := openDebt(t, e, DebtIOwe, "50")
	addPayments(t, e, "debt_payment", "2024-01-10", map[int64]int64{e.usdAcc.ID: -2000, debt.ID: 2000})

	in := debt.Edit()
	in.Principal = decimal.NewFromInt(80)
	got, err := debts.Update(e.ctx, debt.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Owed().Decimal().String() != "60" || got.Principal.Decimal().String() != "80" {
		t.Errorf("principal 50 -> 80 with 20 paid: owed %s, principal %s", got.Owed(), got.Principal)
	}

	in.Direction = DebtOwedToMe
	if _, err := debts.Update(e.ctx, debt.ID, in); err != ErrDebtDirectionLocked {
		t.Errorf("flipping a debt with payments: %v", err)
	}

	fresh := openDebt(t, e, DebtIOwe, "40")
	in = fresh.Edit()
	in.Direction = DebtOwedToMe
	got, err = debts.Update(e.ctx, fresh.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance.Decimal().String() != "40" || got.Owed().Decimal().String() != "40" {
		t.Errorf("a debt without payments can change direction: balance %s", got.Balance)
	}
}

func TestSettlingAndReopeningADebt(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	debt := openDebt(t, e, DebtIOwe, "30")
	if _, err := debts.Reopen(e.ctx, debt.ID); err == nil {
		t.Error("an open debt cannot be reopened")
	}

	addPayments(t, e, "debt_payment", "2024-01-10", map[int64]int64{e.usdAcc.ID: -3000, debt.ID: 3000})
	debts.maybePayOff(e.ctx, debt.ID)
	if d, _ := debts.ByID(e.ctx, debt.ID); !d.IsPaidOff() || !d.Owed().IsZero() {
		t.Fatalf("after the last payment %+v", d)
	}
	if open, _ := debts.All(e.ctx, false); len(open) != 0 {
		t.Errorf("a settled debt is still open: %+v", open)
	}
	if all, _ := debts.All(e.ctx, true); len(all) != 1 {
		t.Errorf("All with completed lists %d debts", len(all))
	}

	reopened, err := debts.Reopen(e.ctx, debt.ID)
	if err != nil || reopened.IsPaidOff() {
		t.Fatalf("reopen: %+v, %v", reopened, err)
	}
}

func TestDebtSummaryFoldsOwedAmounts(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	openDebt(t, e, DebtIOwe, "100")
	openDebt(t, e, DebtOwedToMe, "30")
	jpy, err := debts.Create(e.ctx, DebtInput{Name: "yen", Direction: DebtIOwe, CurrencyID: e.jpy.ID, Amount: decimal.NewFromInt(5000)})
	if err != nil {
		t.Fatal(err)
	}
	addPayments(t, e, "debt_payment", "2024-01-10", map[int64]int64{e.jpyAcc.ID: -1000, jpy.ID: 1000})

	sum := debts.Summary(e.ctx)
	if sum.Count != 3 || sum.IOwe.Decimal().String() != "140" || sum.OwedToMe.Decimal().String() != "30" ||
		sum.Net().Decimal().String() != "-110" {
		t.Errorf("summary %+v (I owe %s, owed to me %s)", sum, sum.IOwe, sum.OwedToMe)
	}
	if want := money.Zero(e.usd.Unit()); sum.Net().Unit() != want.Unit() {
		t.Errorf("the summary is in %v", sum.Net().Unit())
	}
}

func TestDeleteRefusesDebtWithPayments(t *testing.T) {
	e := newMoneyEnv(t)
	debts := newDebts(e)
	accts := Accounts{DB: e.db}
	clean := openDebt(t, e, DebtIOwe, "10")
	used := openDebt(t, e, DebtIOwe, "10")
	addPayments(t, e, "debt_payment", "2024-01-10", map[int64]int64{e.usdAcc.ID: -100, used.ID: 100})

	if err := debts.Delete(e.ctx, used.ID); err == nil {
		t.Error("a debt with payments cannot be deleted")
	}
	if err := debts.Delete(e.ctx, clean.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := accts.ByID(e.ctx, clean.ID); got != nil {
		t.Error("deleted debt still found")
	}
	if err := accts.Delete(e.ctx, e.usdAcc.ID); err == nil {
		t.Error("an account with payments cannot be deleted")
	}
}

func TestAccountsLeaveServiceAccountsToTheirOwners(t *testing.T) {
	e := newMoneyEnv(t)
	accts := Accounts{DB: e.db}
	debt := openDebt(t, e, DebtIOwe, "10")
	spaceID, err := accts.SpaceAccount(e.ctx, e.usd.ID)
	if err != nil {
		t.Fatal(err)
	}

	for _, typ := range []string{"debt", "space"} {
		if _, err := accts.Create(e.ctx, AccountInput{Name: "x", Type: typ, CurrencyID: e.usd.ID, IsActive: true}); err != ErrServiceAccount {
			t.Errorf("create %s: %v", typ, err)
		}
	}
	if _, err := accts.Update(e.ctx, e.usdAcc.ID, AccountInput{Name: "x", Type: "debt", CurrencyID: e.usd.ID}); err != ErrServiceAccount {
		t.Errorf("turning an account into a debt: %v", err)
	}
	for _, id := range []int64{debt.ID, spaceID} {
		if _, err := accts.Update(e.ctx, id, AccountInput{Name: "x", Type: "cash", CurrencyID: e.usd.ID}); err != ErrServiceAccount {
			t.Errorf("update %d: %v", id, err)
		}
		if err := accts.Delete(e.ctx, id); err != ErrServiceAccount {
			t.Errorf("delete %d: %v", id, err)
		}
	}
	if got, _ := newDebts(e).ByID(e.ctx, e.usdAcc.ID); got != nil {
		t.Errorf("an ordinary account is not a debt: %+v", got)
	}
}
