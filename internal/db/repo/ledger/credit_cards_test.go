package ledger_test

import (
	"testing"
	"time"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

func TestCreditCardAndDailyFlowsRoundTrip(t *testing.T) {
	f := newFixture(t)
	unit := money.Unit{ID: f.usd, Decimals: 2}
	id, err := f.repo.InsertAccount(ctx, ledgerrepo.AccountFields{
		Name: "Gold", Type: "credit", CurrencyID: f.usd, IsActive: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := f.repo.CreditCard(ctx, id); c != nil {
		t.Fatalf("card before saving: %+v", c)
	}
	day := 20
	if err := f.repo.SaveCreditCard(ctx, id, ledgerrepo.CreditCard{
		Limit: money.New(150000, unit), Cycle: ledgerrepo.CycleFixedDay, StatementDay: &day, GraceDays: 25,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := f.repo.CreditCard(ctx, id)
	if err != nil || got == nil || got.ID != id || got.Type != "credit" || got.Currency.Code == "" ||
		got.Terms.Limit.Minor() != 150000 || *got.Terms.StatementDay != 20 || got.Terms.PeriodDays != nil ||
		got.Terms.GraceDays != 25 {
		t.Fatalf("card %+v %v", got, err)
	}
	// Saving again replaces the terms.
	day = 5
	if err := f.repo.SaveCreditCard(ctx, id, ledgerrepo.CreditCard{
		Limit: money.New(1, unit), Cycle: ledgerrepo.CycleFixedDay, StatementDay: &day,
	}); err != nil {
		t.Fatal(err)
	}
	all, err := f.repo.CreditCards(ctx, false)
	if err != nil || len(all) != 1 || *all[0].Terms.StatementDay != 5 || all[0].Terms.GraceDays != 0 {
		t.Fatalf("cards %+v %v", all, err)
	}
	if active, _ := f.repo.CreditCards(ctx, true); len(active) != 1 {
		t.Errorf("active cards %+v", active)
	}
	// An ordinary account has no card.
	if c, _ := f.repo.CreditCard(ctx, f.cash); c != nil {
		t.Errorf("card of a cash account: %+v", c)
	}

	f.pay(t, "expense", "confirmed", "2026-01-02", map[int64]int64{id: -3000})
	f.pay(t, "expense", "confirmed", "2026-01-02", map[int64]int64{id: -500})
	f.pay(t, "income", "confirmed", "2026-01-02", map[int64]int64{id: 200})
	f.pay(t, "expense", "pending", "2026-01-03", map[int64]int64{id: -999})
	flows, err := f.repo.DailyFlows(ctx, id)
	want := ledgerrepo.DayFlow{Day: "2026-01-02", Delta: -3300, Inflow: 200, Outflow: -3500}
	if err != nil || len(flows) != 1 || flows[0] != want {
		t.Fatalf("flows %+v %v, want [%+v]", flows, err, want)
	}

	// The payments keep the account: the database refuses to delete it.
	if err := f.repo.DeleteCreditCard(ctx, id); err == nil {
		t.Fatal("deleted a card that has payments")
	}
	if _, err := f.repo.DB.ExecContext(ctx, `DELETE FROM payments WHERE account_id = ?`, id); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.DeleteCreditCard(ctx, id); err != nil {
		t.Fatal(err)
	}
	if c, _ := f.repo.CreditCard(ctx, id); c != nil {
		t.Errorf("card after delete: %+v", c)
	}
	if a, _ := f.repo.Account(ctx, id); a != nil {
		t.Errorf("account after delete: %+v", a)
	}
}
