package ledger

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"savvy-go/internal/audit"
	"savvy-go/internal/money"
)

var recNow = time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

func (e *moneyEnv) recs() Reconciliations { return Reconciliations{DB: e.db} }

func item(acc int64, actual string) ReconcileItem {
	return ReconcileItem{AccountID: acc, Actual: dec(actual)}
}

func (e *moneyEnv) reconcile(t *testing.T, date string, items ...ReconcileItem) []ReconciliationView {
	t.Helper()
	out, err := e.recs().Reconcile(e.ctx, date, items, recNow)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (e *moneyEnv) recRows(t *testing.T) []audit.Entry {
	t.Helper()
	return e.entries(t, audit.Filter{EntityType: EntityReconciliation})
}

func (e *moneyEnv) pointCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM reconciliations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestReconcileRejectsBadInputAndStoresNothing(t *testing.T) {
	e := pointEnv(t)
	svc := e.recs()
	debt := openDebt(t, e, DebtIOwe, "50").ID
	cases := []struct {
		name  string
		date  string
		items []ReconcileItem
		want  error
	}{
		{"no items", "2024-01-31", nil, ErrNoReconcileItems},
		{"duplicate account", "2024-01-31", []ReconcileItem{item(e.usdAcc.ID, "1"), item(e.usdAcc.ID, "2")}, ErrDuplicateAccount},
		{"future", "2024-03-02", []ReconcileItem{item(e.usdAcc.ID, "1")}, ErrReconcileFuture},
		{"bad date", "2024-13-01", []ReconcileItem{item(e.usdAcc.ID, "1")}, ErrReconcileDate},
		{"empty date", "", []ReconcileItem{item(e.usdAcc.ID, "1")}, ErrReconcileDate},
		{"unknown account", "2024-01-31", []ReconcileItem{item(e.usdAcc.ID, "1"), item(9999, "1")}, ErrUnknownAccount},
		{"service account", "2024-01-31", []ReconcileItem{item(e.usdAcc.ID, "1"), item(debt, "1")}, ErrServiceAccount},
		{"too many decimals", "2024-01-31", []ReconcileItem{item(e.usdAcc.ID, "1"), item(e.jpyAcc.ID, "1.5")}, ErrReconcileScale},
		{"overflow", "2024-01-31", []ReconcileItem{item(e.usdAcc.ID, "99999999999999999999")}, money.ErrOutOfRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Reconcile(e.ctx, c.date, c.items, recNow)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if n := e.pointCount(t); n != 0 {
				t.Errorf("%d points stored", n)
			}
			if rows := e.recRows(t); len(rows) != 0 {
				t.Errorf("%d audit rows", len(rows))
			}
		})
	}
}

func TestReconcileTodayIsNotFuture(t *testing.T) {
	e := pointEnv(t)
	e.reconcile(t, "2024-03-01", item(e.usdAcc.ID, "1"))
}

func TestReconcileBatchSharesOneAuditBatch(t *testing.T) {
	e := pointEnv(t)
	note := "statement"
	out, err := e.recs().Reconcile(e.agent(3), "2024-01-31",
		[]ReconcileItem{{AccountID: e.usdAcc.ID, Actual: dec("200"), Note: &note}, item(e.jpyAcc.ID, "2500")}, recNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].AccountID != e.usdAcc.ID || out[1].AccountID != e.jpyAcc.ID {
		t.Fatalf("out = %+v", out)
	}
	// usd: 100 + 50 - 20 - 5 = 125 expected on 01-31; actual 200.
	if out[0].Expected.Decimal().String() != "125" || out[0].Difference.Decimal().String() != "75" ||
		out[0].Note == nil || *out[0].Note != "statement" {
		t.Errorf("usd view = %+v", out[0])
	}
	rows := e.recRows(t)
	if len(rows) != 2 || rows[0].BatchID == "" || rows[0].BatchID != rows[1].BatchID {
		t.Fatalf("rows = %+v", rows)
	}
	for _, r := range rows {
		if r.Op != audit.OpCreate || r.Before != nil || r.After == nil || r.TokenID == nil || *r.TokenID != 3 {
			t.Errorf("row = %+v", r)
		}
	}
}

func TestReconcileSameDateReplacesAndAuditsUpdate(t *testing.T) {
	e := pointEnv(t)
	first := e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "200"))[0]
	second := e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "210"))[0]
	if second.ID != first.ID || e.pointCount(t) != 1 || second.Actual.Decimal().String() != "210" {
		t.Fatalf("first %+v second %+v", first, second)
	}
	rows := e.recRows(t) // newest first
	if len(rows) != 2 || rows[0].Op != audit.OpUpdate || rows[0].Before == nil || rows[0].After == nil {
		t.Fatalf("rows = %+v", rows)
	}
	// The same values again change nothing and log nothing.
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "210"))
	if len(e.recRows(t)) != 2 {
		t.Errorf("a no-op write was logged")
	}
}

func TestReconcileListFiltersOrderAndChecks(t *testing.T) {
	e := pointEnv(t)
	e.reconcile(t, "2024-01-15", item(e.usdAcc.ID, "200"), item(e.jpyAcc.ID, "3100"))
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "180"))
	all, err := e.recs().List(e.ctx, ReconciliationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Date != "2024-01-31" || all[1].AccountID != e.usdAcc.ID || all[2].AccountID != e.jpyAcc.ID {
		t.Fatalf("order: %+v", all)
	}
	// 01-15: expected 100+50-20 = 130, actual 200. 01-31: from anchor 200, -5 on 01-20 = 195; actual 180.
	if all[2].Expected.Decimal().String() != "3000" || all[2].Difference.Decimal().String() != "100" {
		t.Errorf("jpy = %+v", all[2])
	}
	if all[1].Expected.Decimal().String() != "130" || all[1].Difference.Decimal().String() != "70" {
		t.Errorf("usd first = %+v", all[1])
	}
	if all[0].Expected.Decimal().String() != "195" || all[0].Difference.Decimal().String() != "-15" {
		t.Errorf("usd second measured against the first = %+v", all[0])
	}
	// A range keeps the expected value computed from the full history.
	late, _ := e.recs().List(e.ctx, ReconciliationFilter{From: "2024-01-20"})
	if len(late) != 1 || late[0].Expected.Decimal().String() != "195" {
		t.Errorf("late = %+v", late)
	}
	early, _ := e.recs().List(e.ctx, ReconciliationFilter{To: "2024-01-20", AccountIDs: []int64{e.usdAcc.ID}})
	if len(early) != 1 || early[0].Date != "2024-01-15" {
		t.Errorf("early = %+v", early)
	}
	none, err := e.recs().List(e.ctx, ReconciliationFilter{AccountIDs: []int64{e.btcAcc.ID}})
	if err != nil || len(none) != 0 {
		t.Errorf("none = %+v, %v", none, err)
	}
}

func TestReconciliationDifferenceInBaseCurrency(t *testing.T) {
	e := pointEnv(t)
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "125"), item(e.jpyAcc.ID, "3500")) // usd 0 off, jpy +500 = 5 USD
	list, _ := e.recs().List(e.ctx, ReconciliationFilter{})
	byAcc := map[int64]ReconciliationView{}
	for _, v := range list {
		byAcc[v.AccountID] = v
	}
	if d := byAcc[e.usdAcc.ID].DifferenceBase; d == nil || !d.Decimal().IsZero() || d.Unit() != e.usd.Unit() {
		t.Errorf("usd base = %v", d)
	}
	if d := byAcc[e.jpyAcc.ID].DifferenceBase; d == nil || d.Decimal().String() != "5" || d.Unit() != e.usd.Unit() {
		t.Errorf("jpy base = %v", d)
	}
	// A currency with no rate has no base figure.
	if _, err := e.db.Exec(`UPDATE currencies SET rate = 0 WHERE id = ?`, e.jpy.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = e.recs().List(e.ctx, ReconciliationFilter{AccountIDs: []int64{e.jpyAcc.ID}})
	if len(list) != 1 || list[0].DifferenceBase != nil {
		t.Errorf("no rate: %+v", list)
	}
}

func TestDayReconciliationsSumBaseDifferences(t *testing.T) {
	e := pointEnv(t)
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "135"), item(e.jpyAcc.ID, "3500")) // +10 usd, +500 jpy = +5
	e.reconcile(t, "2024-02-10", item(e.usdAcc.ID, "135"))                            // 0 off
	days, err := e.recs().DayReconciliations(e.ctx, "2024-01-01", "2024-02-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 1 || days[0].Date != "2024-01-31" || len(days[0].Points) != 2 ||
		days[0].DifferenceBase.Decimal().String() != "15" || days[0].Unconverted {
		t.Fatalf("days = %+v", days)
	}
}

func TestDeleteReconciliationIsAudited(t *testing.T) {
	e := pointEnv(t)
	v := e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "200"))[0]
	if err := e.recs().Delete(e.ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	if e.pointCount(t) != 0 {
		t.Fatal("point stays")
	}
	rows := e.recRows(t)
	if len(rows) != 2 || rows[0].Op != audit.OpDelete || rows[0].Before == nil || rows[0].After != nil {
		t.Fatalf("rows = %+v", rows)
	}
	if err := e.recs().Delete(e.ctx, v.ID); !errors.Is(err, ErrUnknownReconciliation) {
		t.Errorf("second delete = %v", err)
	}
}

func (e *moneyEnv) pointOf(t *testing.T, id int64) *ReconciliationView {
	t.Helper()
	list, err := e.recs().List(e.ctx, ReconciliationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range list {
		if v.ID == id {
			return &v
		}
	}
	return nil
}

func TestRollbackReconciliationCreateUpdateDelete(t *testing.T) {
	e := pointEnv(t)
	v := e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "200"))[0]
	note := "n"
	if _, err := e.recs().Reconcile(e.ctx, "2024-01-31", []ReconcileItem{{AccountID: e.usdAcc.ID, Actual: dec("210"), Note: &note}}, recNow); err != nil {
		t.Fatal(err)
	}
	// Update rolled back: previous balance and note return.
	rows := e.recRows(t)
	if _, err := e.rollbacks().Undo(e.ctx, []int64{rows[0].ID}, 0); err != nil {
		t.Fatal(err)
	}
	got := e.pointOf(t, v.ID)
	if got == nil || got.Actual.Decimal().String() != "200" || got.Note != nil {
		t.Fatalf("after undo update: %+v", got)
	}
	// Create rolled back: the point goes.
	if _, err := e.rollbacks().Undo(e.ctx, []int64{rows[1].ID}, 0); err != nil {
		t.Fatal(err)
	}
	if e.pointCount(t) != 0 {
		t.Fatal("create not undone")
	}
	// Delete rolled back: re-created under its id.
	v2 := e.reconcile(t, "2024-01-20", item(e.usdAcc.ID, "300"))[0]
	if err := e.recs().Delete(e.ctx, v2.ID); err != nil {
		t.Fatal(err)
	}
	del := e.recRows(t)[0]
	if _, err := e.rollbacks().Undo(e.ctx, []int64{del.ID}, 0); err != nil {
		t.Fatal(err)
	}
	back := e.pointOf(t, v2.ID)
	if back == nil || back.Actual.Decimal().String() != "300" || back.Date != "2024-01-20" {
		t.Fatalf("after undo delete: %+v", back)
	}
}

func TestRollbackDeleteRefusedWhenDateTaken(t *testing.T) {
	e := pointEnv(t)
	v := e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "200"))[0]
	_ = e.recs().Delete(e.ctx, v.ID)
	del := e.recRows(t)[0]
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "250"))
	if _, err := e.rollbacks().Undo(e.ctx, []int64{del.ID}, 0); !errors.Is(err, audit.ErrNotUndoable) {
		t.Fatalf("err = %v", err)
	}
	if e.pointCount(t) != 1 {
		t.Error("points changed")
	}
}

func TestRollbackWholeReconcileBatch(t *testing.T) {
	e := pointEnv(t)
	e.reconcile(t, "2024-01-31", item(e.usdAcc.ID, "200"), item(e.jpyAcc.ID, "3100"))
	rows := e.recRows(t)
	ids, err := e.rollbacks().UndoBatch(e.ctx, rows[0].BatchID, 0)
	if err != nil || len(ids) != 2 {
		t.Fatalf("ids %v err %v", ids, err)
	}
	if e.pointCount(t) != 0 {
		t.Error("batch not undone")
	}
	var snap struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(rows[0].After, &snap); err != nil || snap.Date != "2024-01-31" {
		t.Errorf("snapshot = %s", rows[0].After)
	}
}
