package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"savvy-go/internal/audit"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/store"
)

func i64(v int64) *int64 { return &v }

// agent is the actor of an MCP token: user 1, token tok.
func (e *moneyEnv) agent(tok int64) context.Context {
	return audit.WithActor(e.ctx, audit.Actor{UserID: i64(1), TokenID: i64(tok), TokenLabel: "agent", Source: audit.SourceMCP})
}

// person is user 1 in the browser: no token.
func (e *moneyEnv) person() context.Context {
	return audit.WithActor(e.ctx, audit.Actor{UserID: i64(1), Source: audit.SourceUI})
}

func (e *moneyEnv) rollbacks() Rollbacks { return Rollbacks{DB: e.db} }

func (e *moneyEnv) entries(t *testing.T, f audit.Filter) []audit.Entry {
	t.Helper()
	list, err := audit.Log{DB: e.db}.List(e.ctx, f, 100)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func (e *moneyEnv) dated(typ string, acc int64, amount string) TxInput {
	in := simple(typ, acc, amount)
	d := "2024-02-01"
	in.Date = &d
	return in
}

// Every change is a row of the log, written with the change: the actor, the
// door and the exact money after.
func TestCreateIsLoggedWithActorAndExactMoney(t *testing.T) {
	e := newMoneyEnv(t)
	tx, err := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "12.34"))
	if err != nil {
		t.Fatal(err)
	}
	rows := e.entries(t, audit.Filter{TokenID: 7})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Op != audit.OpCreate || r.EntityType != EntityTransaction || *r.EntityID != tx.ID || r.Source != audit.SourceMCP ||
		*r.UserID != 1 || r.TokenLabel != "agent" || r.Before != nil {
		t.Fatalf("row = %+v", r)
	}
	var after struct {
		Payments []struct {
			AccountID int64 `json:"account_id"`
			Minor     int64 `json:"minor"`
		}
	}
	if err := json.Unmarshal(r.After, &after); err != nil {
		t.Fatal(err)
	}
	if len(after.Payments) != 1 || after.Payments[0].Minor != -1234 || after.Payments[0].AccountID != e.usdAcc.ID {
		t.Errorf("after = %s, want one payment of -1234 minor units on the usd account", r.After)
	}
}

// A change with no request behind it is the system's: no user, no token.
func TestChangeWithoutActorIsSystem(t *testing.T) {
	e := newMoneyEnv(t)
	if _, err := e.txs.Create(e.ctx, e.dated("income", e.usdAcc.ID, "5")); err != nil {
		t.Fatal(err)
	}
	r := e.entries(t, audit.Filter{})[0]
	if r.Source != audit.SourceSystem || r.UserID != nil || r.TokenID != nil {
		t.Fatalf("row = %+v, want system with no user and no token", r)
	}
}

// The case of the issue: an agent that cannot delete takes back its own create.
func TestRollbackOfCreateDeletesTheTransaction(t *testing.T) {
	e := newMoneyEnv(t)
	tx, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "30"))
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("70")) {
		t.Fatalf("balance = %s", got)
	}
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.txs.ByID(e.ctx, tx.ID); got != nil {
		t.Errorf("transaction still there: %+v", got)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("100")) {
		t.Errorf("balance = %s, want 100 back", got)
	}
	if again, err := e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7); !errors.Is(err, audit.ErrAlreadyRolledBack) {
		t.Errorf("second rollback = %v, %v, want ErrAlreadyRolledBack", again, err)
	}
}

func TestRollbackOfUpdateRestoresMoneyAndDetails(t *testing.T) {
	e := newMoneyEnv(t)
	tx, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "30"))
	upd := e.dated("expense", e.usdAcc.ID, "45.50")
	upd.Description = ptr("oops")
	if _, err := e.txs.Update(e.agent(7), tx.ID, upd); err != nil {
		t.Fatal(err)
	}
	if got := e.balance(t, e.usdAcc.ID); !got.Equal(dec("54.5")) {
		t.Fatalf("balance = %s", got)
	}
	rows := e.entries(t, audit.Filter{TokenID: 7})
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{rows[0].ID}, 7); err != nil {
		t.Fatal(err)
	}
	got, _ := e.txs.ByID(e.ctx, tx.ID)
	if got == nil || got.Description != nil || !e.balance(t, e.usdAcc.ID).Equal(dec("70")) {
		t.Errorf("tx = %+v balance = %s, want the 30 expense without description", got, e.balance(t, e.usdAcc.ID))
	}
}

func TestRollbackOfDeleteRestoresUnderItsId(t *testing.T) {
	e := newMoneyEnv(t)
	in := e.dated("expense", e.usdAcc.ID, "30")
	in.Description = ptr("lunch")
	tx, _ := e.txs.Create(e.ctx, in)
	// The agent deletes what a person made; its own delete is what it can undo.
	if err := e.txs.Delete(e.agent(7), tx.ID); err != nil {
		t.Fatal(err)
	}
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7); err != nil {
		t.Fatal(err)
	}
	got, _ := e.txs.ByID(e.ctx, tx.ID)
	if got == nil || got.Description == nil || *got.Description != "lunch" || !e.balance(t, e.usdAcc.ID).Equal(dec("70")) {
		t.Fatalf("restored = %+v, balance %s", got, e.balance(t, e.usdAcc.ID))
	}
}

// What a person did after the agent is theirs: the agent may not undo under it.
func TestRollbackRefusedWhenSomeoneChangedItAfter(t *testing.T) {
	e := newMoneyEnv(t)
	tx, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "30"))
	d := "improved"
	if _, err := e.txs.Update(e.person(), tx.ID, TxInput{Description: &d}); err != nil {
		t.Fatal(err)
	}
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	_, err := e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7)
	var ov *audit.OverlapError
	if !errors.As(err, &ov) || ov.EntityID != tx.ID {
		t.Fatalf("err = %v, want an overlap on transaction %d", err, tx.ID)
	}
	if got, _ := e.txs.ByID(e.ctx, tx.ID); got == nil {
		t.Error("the transaction was deleted despite the refusal")
	}
}

// The agent's own later change stands in the way as well, until it is undone first.
func TestRollbackOfAChainGoesNewestFirst(t *testing.T) {
	e := newMoneyEnv(t)
	tx, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "30"))
	if _, err := e.txs.Update(e.agent(7), tx.ID, e.dated("expense", e.usdAcc.ID, "40")); err != nil {
		t.Fatal(err)
	}
	rows := e.entries(t, audit.Filter{TokenID: 7}) // update, create
	create, update := rows[1], rows[0]
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{create.ID}, 7); err == nil {
		t.Fatal("the create was undone under its own update")
	}
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{update.ID}, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{create.ID}, 7); err != nil {
		t.Fatalf("after the update was undone: %v", err)
	}
	// Both together, in one call: newest first is done for the caller.
	tx2, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "10"))
	_, _ = e.txs.Update(e.agent(7), tx2.ID, e.dated("expense", e.usdAcc.ID, "11"))
	both := e.entries(t, audit.Filter{TokenID: 7, EntityID: tx2.ID})
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{both[1].ID, both[0].ID}, 7); err != nil {
		t.Fatalf("both in one call: %v", err)
	}
}

// A token undoes only its own rows; another token's look like they do not exist.
func TestRollbackIsPerToken(t *testing.T) {
	e := newMoneyEnv(t)
	_, _ = e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "30"))
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	if _, err := e.rollbacks().Undo(e.agent(8), []int64{row.ID}, 8); !errors.Is(err, audit.ErrNotFound) {
		t.Fatalf("other token: %v, want ErrNotFound", err)
	}
	if _, err := e.rollbacks().Undo(e.person(), []int64{row.ID}, 0); err != nil {
		t.Fatalf("no token restriction (a space admin in the browser): %v", err)
	}
}

// A bulk create is one batch: one call takes it all back, or none.
func TestRollbackOfABatch(t *testing.T) {
	e := newMoneyEnv(t)
	ids, err := e.txs.AddMany(e.agent(7), []TxInput{
		e.dated("expense", e.usdAcc.ID, "1"), e.dated("expense", e.usdAcc.ID, "2"), e.dated("expense", e.usdAcc.ID, "3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	rows := e.entries(t, audit.Filter{TokenID: 7})
	if len(rows) != 3 || rows[0].BatchID == "" || rows[0].BatchID != rows[2].BatchID {
		t.Fatalf("rows = %+v, want three sharing a batch", rows)
	}
	// One of the three was edited by a person: the whole batch is refused.
	d := "x"
	_, _ = e.txs.Update(e.person(), ids[1], TxInput{Description: &d})
	if _, err := e.rollbacks().UndoBatch(e.agent(7), rows[0].BatchID, 7); err == nil {
		t.Fatal("batch undone under a person's edit")
	}
	for _, id := range ids {
		if got, _ := e.txs.ByID(e.ctx, id); got == nil {
			t.Fatalf("transaction %d was deleted by a refused batch rollback", id)
		}
	}
	// Fix it by hand: undo the person's edit is not the agent's call; use a fresh batch.
	ids, _ = e.txs.AddMany(e.agent(7), []TxInput{e.dated("expense", e.usdAcc.ID, "4"), e.dated("expense", e.usdAcc.ID, "5")})
	batch := e.entries(t, audit.Filter{TokenID: 7, EntityID: ids[0]})[0].BatchID
	if _, err := e.rollbacks().UndoBatch(e.agent(7), batch, 7); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if got, _ := e.txs.ByID(e.ctx, id); got != nil {
			t.Errorf("transaction %d survived its batch's rollback", id)
		}
	}
}

// The external key is not the log's business, but a restore must not make a duplicate of it.
func TestRollbackOfDeleteRefusedWhenKeyTaken(t *testing.T) {
	e := newMoneyEnv(t)
	src, key := audit.SourceMCP, "R-1"
	in := e.dated("expense", e.usdAcc.ID, "30")
	in.ExternalSource, in.ExternalID = &src, &key
	tx, err := e.txs.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.txs.Delete(e.agent(7), tx.ID); err != nil {
		t.Fatal(err)
	}
	again := e.dated("expense", e.usdAcc.ID, "30")
	again.ExternalSource, again.ExternalID = &src, &key
	if _, err := e.txs.Create(e.ctx, again); err != nil {
		t.Fatal(err)
	}
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	_, err = e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7)
	if !errors.Is(err, ErrExternalIDExists) {
		t.Fatalf("err = %v, want the external id refusal", err)
	}
}

// Cleanup is manual, by a signed-in user, and leaves one row saying who and when.
func TestClearLeavesOneRowAboutTheCleanup(t *testing.T) {
	e := newMoneyEnv(t)
	_, _ = e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "1"))
	_, _ = e.txs.Create(e.agent(8), e.dated("expense", e.usdAcc.ID, "2"))
	log := audit.Log{DB: e.db}
	if _, err := log.Clear(e.ctx, 0); !errors.Is(err, audit.ErrNeedsUser) {
		t.Fatalf("clear by the system = %v, want ErrNeedsUser", err)
	}
	// Only one token's rows.
	if n, err := log.Clear(e.person(), 7); err != nil || n != 1 {
		t.Fatalf("clear token 7 = %d, %v", n, err)
	}
	if n, err := log.Clear(e.person(), 0); err != nil || n != 2 { // token 8's row and the first cleanup row
		t.Fatalf("clear all = %d, %v", n, err)
	}
	rows := e.entries(t, audit.Filter{})
	if len(rows) != 1 || rows[0].Op != audit.OpCleanup || *rows[0].UserID != 1 || rows[0].EntityID != nil ||
		!strings.Contains(rows[0].Details, "removed 2") {
		t.Fatalf("rows = %+v, want one cleanup row by user 1", rows)
	}
}

// Rows that only came to be by a rollback are not rolled back again (undo of an undo is a new change).
func TestRollbackRowsAreNotUndoable(t *testing.T) {
	e := newMoneyEnv(t)
	_, _ = e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "1"))
	row := e.entries(t, audit.Filter{TokenID: 7})[0]
	_, _ = e.rollbacks().Undo(e.agent(7), []int64{row.ID}, 7)
	rb := e.entries(t, audit.Filter{TokenID: 7})[0]
	if rb.Op != audit.OpRollback {
		t.Fatalf("newest row = %+v", rb)
	}
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{rb.ID}, 7); !errors.Is(err, audit.ErrNotUndoable) {
		t.Errorf("undo of a rollback = %v, want ErrNotUndoable", err)
	}
}

// Putting a deleted transaction back fails, and keeps nothing, when the
// account it was on is gone since: the error says what is missing. It runs on
// the real store, which is what turns the driver's error into ErrConstraint.
func TestRollbackOfDeleteFailsOnMissingAccount(t *testing.T) {
	ctx := context.Background()
	d := dbtest.Space(t)
	cur, err := Currencies{DB: d}.Create(ctx, Currency{Code: "JPY", Name: "JPY", Symbol: "¥", Decimals: 0, Rate: dec("0.01")})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := Accounts{DB: d}.Create(ctx, AccountInput{Name: "jpy", Type: "cash", CurrencyID: cur.ID, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	agent := audit.WithActor(ctx, audit.Actor{UserID: i64(1), TokenID: i64(7), TokenLabel: "agent", Source: audit.SourceMCP})
	txs := Transactions{DB: d}
	in := simple("expense", acc.ID, "300")
	day := "2024-02-01"
	in.Date = &day
	tx, err := txs.Create(agent, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := txs.Delete(agent, tx.ID); err != nil {
		t.Fatal(err)
	}
	if err := (Accounts{DB: d}).Delete(ctx, acc.ID); err != nil {
		t.Fatal(err)
	}
	log := audit.Log{DB: d}
	rows, _ := log.List(ctx, audit.Filter{TokenID: 7}, 10)
	del := rows[0]

	_, err = Rollbacks{DB: d}.Undo(agent, []int64{del.ID}, 7)
	if !errors.Is(err, store.ErrConstraint) || !strings.Contains(err.Error(), "is gone") {
		t.Fatalf("err = %v, want a constraint error naming what is gone", err)
	}
	if got, _ := txs.ByID(ctx, tx.ID); got != nil {
		t.Errorf("a half-restored transaction is left: %+v", got)
	}
	after, _ := log.List(ctx, audit.Filter{}, 10)
	if len(after) != len(rows) || after[0].RolledBackBy != nil {
		t.Errorf("the failed rollback left rows behind: %d -> %d, newest %+v", len(rows), len(after), after[0])
	}
}

// The batched check answers as the one-by-one check does, for every kind of answer.
func TestUndoableAllMatchesUndoable(t *testing.T) {
	e := newMoneyEnv(t)
	a, _ := e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "1"))
	_, _ = e.txs.Create(e.agent(7), e.dated("expense", e.usdAcc.ID, "2"))
	_, _ = e.txs.Create(e.agent(8), e.dated("expense", e.usdAcc.ID, "3"))
	d := "again"
	_, _ = e.txs.Update(e.agent(7), a.ID, TxInput{Description: &d}) // overlaps a's create
	rows := e.entries(t, audit.Filter{})
	if _, err := e.rollbacks().Undo(e.agent(7), []int64{rows[0].ID}, 7); err != nil { // an undone row and a rollback row
		t.Fatal(err)
	}
	rows = e.entries(t, audit.Filter{})

	for _, tok := range []int64{0, 7, 8} {
		got, err := e.rollbacks().UndoableAll(e.ctx, rows, tok)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range rows {
			want := e.rollbacks().Undoable(e.ctx, r, tok)
			if fmt.Sprint(got[i]) != fmt.Sprint(want) {
				t.Errorf("token %d, entry %d (%s): batched %v, single %v", tok, r.ID, r.Op, got[i], want)
			}
		}
	}
}

// A bulk call larger than one read chunk still logs every transaction, in
// order, as one batch, with the same snapshot a single create writes.
func TestBigBulkCreateLogsEveryTransaction(t *testing.T) {
	e := newMoneyEnv(t)
	ins := make([]TxInput, 1100)
	for i := range ins {
		ins[i] = e.dated("expense", e.usdAcc.ID, "1.25")
	}
	ids, err := e.txs.AddMany(e.agent(7), ins)
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := audit.Log{DB: e.db}.List(e.ctx, audit.Filter{TokenID: 7}, 2000)
	if len(rows) != len(ids) {
		t.Fatalf("%d rows for %d transactions", len(rows), len(ids))
	}
	for i, r := range rows { // newest first
		if *r.EntityID != ids[len(ids)-1-i] || r.BatchID != rows[0].BatchID || r.BatchID == "" || !strings.Contains(string(r.After), `"minor":-125`) {
			t.Fatalf("row %d = %+v", i, r)
		}
	}
	single, _ := e.txs.Create(e.agent(8), e.dated("expense", e.usdAcc.ID, "1.25"))
	one := e.entries(t, audit.Filter{TokenID: 8})[0]
	var a, b map[string]any
	_ = json.Unmarshal(one.After, &a)
	_ = json.Unmarshal(rows[0].After, &b)
	for _, k := range []string{"id", "created_at", "updated_at", "day_order"} {
		delete(a, k)
		delete(b, k)
	}
	if fmt.Sprint(a) != fmt.Sprint(b) || single == nil {
		t.Errorf("bulk snapshot %v differs from single %v", b, a)
	}
}
