package transfers_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	transfersrepo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/money"
)

var ctx = context.Background()

func TestLinksAndTrustedKeysOnTheServer(t *testing.T) {
	d := dbtest.Server(t)
	users := authrepo.Repo{DB: d}
	now := time.Now().UTC()
	ann, _ := users.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", now)
	// Two registered spaces to link.
	for _, uuid := range []string{"a", "b"} {
		if _, err := d.ExecContext(ctx, `INSERT INTO spaces (uuid, name, created_by) VALUES (?, ?, ?)`, uuid, uuid, ann); err != nil {
			t.Fatal(err)
		}
	}
	s := transfersrepo.Server{DB: d}

	if ok, _ := s.Linked(ctx, 1, 2); ok {
		t.Fatal("linked before linking")
	}
	if err := s.InsertLink(ctx, 1, 2, ann, now); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Linked(ctx, 1, 2); !ok {
		t.Fatal("not linked after linking")
	}
	if links, _ := s.Links(ctx, 2); len(links) != 1 || links[0].A != 1 || links[0].B != 2 {
		t.Fatalf("links = %+v", links)
	}
	if all, _ := s.AllLinks(ctx); len(all) != 1 {
		t.Fatalf("all links = %+v", all)
	}
	if ok, _ := s.DeleteLink(ctx, 1, 2); !ok {
		t.Fatal("unlink failed")
	}
	if ok, _ := s.DeleteLink(ctx, 1, 2); ok {
		t.Fatal("unlinking twice must report that nothing was linked")
	}

	if _, ok := s.PublicKey(ctx, "kid"); ok {
		t.Fatal("an unknown key is trusted")
	}
	if err := s.InsertTrustedKey(ctx, "kid", "cHVi", "Other server", ann, now); err != nil {
		t.Fatal(err)
	}
	if key, ok := s.PublicKey(ctx, "kid"); !ok || key != "cHVi" {
		t.Fatalf("key = %q, %v", key, ok)
	}
	if keys, _ := s.TrustedKeys(ctx); len(keys) != 1 || keys[0].Name != "Other server" || keys[0].CreatedAt == nil {
		t.Fatalf("keys = %+v", keys)
	}
	if ok, _ := s.DeleteTrustedKey(ctx, "kid"); !ok {
		t.Fatal("untrust failed")
	}
	if ok, _ := s.DeleteTrustedKey(ctx, "kid"); ok {
		t.Fatal("untrusting twice must report that nothing was trusted")
	}
}

func record(uuid string) transfersrepo.Record {
	desc := "rent"
	creator := int64(7)
	return transfersrepo.Record{
		UUID: uuid, Date: "2026-03-04", Description: &desc, CreatedBy: &creator, Version: 2, SigV: 1,
		SignerKID: "kid", Signature: "sig", Status: "ok",
		From: transfersrepo.Side{SpaceUUID: "a", AccountID: 1, Amount: 1000, Currency: "USD", Decimals: 2},
		To:   transfersrepo.Side{SpaceUUID: "b", AccountID: 5, Amount: 90000, Currency: "JPY", Decimals: 0},
	}
}

func TestRecordsRoundTripAndKeepTheirRemoteCopy(t *testing.T) {
	r := transfersrepo.Space{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	if rec, err := r.Record(ctx, "nope"); rec != nil || err != nil {
		t.Fatalf("unknown record = %v, %v", rec, err)
	}
	in := record("t1")
	review := "changed_remote"
	in.Review, in.Remote = &review, `{"uuid":"t1"}`
	if err := r.Save(ctx, in, now); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Record(ctx, "t1")
	if got == nil || got.From != in.From || got.To != in.To || got.Version != 2 || got.Review == nil || *got.Review != review ||
		got.Remote != `{"uuid":"t1"}` || got.Description == nil ||
		*got.Description != "rent" ||
		got.DeletedAt != nil {
		t.Fatalf("record = %+v", got)
	}

	// Saving again replaces, and an empty remote clears it.
	in.Version, in.Remote, in.Review = 3, "", nil
	if err := r.Save(ctx, in, now); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Record(ctx, "t1")
	if got.Version != 3 || got.Remote != "" || got.Review != nil {
		t.Fatalf("after replacing = %+v", got)
	}
	if all, _ := r.Records(ctx); len(all) != 1 {
		t.Fatalf("records = %d", len(all))
	}
}

func TestDeletingARecordDeletesItsTransaction(t *testing.T) {
	d := dbtest.Space(t)
	r := transfersrepo.Space{DB: d}
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	usd, _ := l.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "USD", Name: "Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)},
		now,
	)
	acc, _ := l.InsertAccount(
		ctx,
		ledgerrepo.AccountFields{
			Name:           "Cash",
			Type:           "cash",
			CurrencyID:     usd,
			InitialBalance: money.New(0, money.Unit{ID: usd, Decimals: 2}),
			IsActive:       true,
		},
		now,
	)

	if got, err := r.AccountCurrency(ctx, 999); got != nil || err != nil {
		t.Fatalf("unknown account = %v, %v", got, err)
	}
	if got, _ := r.AccountCurrency(ctx, acc); got == nil || got.Code != "USD" || got.Decimals != 2 {
		t.Fatalf("account currency = %+v", got)
	}

	_ = r.Save(ctx, record("t1"), now)
	tx := transfersrepo.Transaction{
		UUID:      "t1",
		Type:      "transfer_out",
		Date:      "2026-03-04",
		Status:    "confirmed",
		AccountID: acc,
		Amount:    1000,
	}
	if has, _ := r.HasTransaction(ctx, "t1"); has {
		t.Fatal("transaction before insert")
	}
	if err := r.InsertTransaction(ctx, tx, now); err != nil {
		t.Fatal(err)
	}
	tx.Amount = 2000
	if err := r.UpdateTransaction(ctx, tx, now); err != nil {
		t.Fatal(err)
	}
	if has, _ := r.HasTransaction(ctx, "t1"); !has {
		t.Fatal("transaction missing")
	}
	if uuid := l.TransferUUID(ctx, 1); uuid != "t1" {
		t.Fatalf("the transaction is not tied to its transfer: %q", uuid)
	}

	if err := r.Delete(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	if rec, _ := r.Record(ctx, "t1"); rec != nil {
		t.Fatal("the record survived")
	}
	if has, _ := r.HasTransaction(ctx, "t1"); has {
		t.Fatal("the transaction survived")
	}
}
