package transfers_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/db/filter"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	transfersrepo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/money"
)

func ptr[T any](v T) *T { return &v }

const payload = `{"uuid":"t1","from":{"space":"a","account":1,"amount":1000,"currency":"USD","decimals":2},` +
	`"to":{"space":"b","account":5,"amount":90000,"currency":"JPY","decimals":0},"date":"2026-03-04",` +
	`"description":"rent","created_by":7,"version":2,"deleted_at":null}`

func TestRecordKeepsItsBytesAndReadsTheGeneratedColumns(t *testing.T) {
	r := transfersrepo.Space{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	if rec, err := r.Record(ctx, "nope"); rec != nil || err != nil {
		t.Fatalf("unknown record = %v, %v", rec, err)
	}
	in := transfersrepo.Record{
		UUID: "t1", Payload: ptr(payload), SigV: 1, SourceKID: ptr("kid"), SourceSig: ptr("sig"), Status: "confirmed",
	}
	if err := r.Save(ctx, in, now); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Record(ctx, "t1")
	if got == nil || got.Payload == nil || *got.Payload != payload || got.Source != "a" || got.Destination != "b" ||
		got.Version != 2 || got.SigV != 1 || *got.SourceKID != "kid" || *got.SourceSig != "sig" ||
		got.DestinationKID != nil || got.Remote != nil {
		t.Fatalf("record = %+v", got)
	}

	// A record only the other space has: no payload, the generated columns read the remote.
	remote := `{"uuid":"t2","from":{"space":"b"},"to":{"space":"a"},"date":"2026-04-01","version":5}`
	if err := r.Save(ctx, transfersrepo.Record{UUID: "t2", SigV: 1, Status: "in_review", Remote: &remote}, now); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Record(ctx, "t2")
	if got.Payload != nil || got.Source != "b" || got.Destination != "a" || got.Version != 5 || *got.Remote != remote {
		t.Fatalf("record from the remote = %+v", got)
	}

	// Saving again replaces, and a nil remote clears it.
	in.Status, in.Remote = "in_review", &remote
	_ = r.Save(ctx, in, now)
	in.Status, in.Remote = "confirmed", nil
	if err := r.Save(ctx, in, now); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Record(ctx, "t1")
	if got.Status != "confirmed" || got.Remote != nil {
		t.Fatalf("after replacing = %+v", got)
	}
	// Newest first, by the date in the record.
	if all, _ := r.Records(ctx); len(all) != 2 || all[0].UUID != "t2" {
		t.Fatalf("records = %+v", all)
	}
}

type books struct {
	repo        transfersrepo.Space
	ledger      ledgerrepo.Repo
	eur, rsd    int64
	card, dinar int64
	now         time.Time
}

func newBooks(t *testing.T) *books {
	t.Helper()
	d := dbtest.Space(t)
	b := &books{repo: transfersrepo.Space{DB: d}, ledger: ledgerrepo.Repo{DB: d}, now: time.Now().UTC()}
	one := decimal.NewFromInt(1)
	var err error
	if b.eur, err = b.ledger.InsertCurrency(ctx,
		ledgerrepo.Currency{Code: "EUR", Name: "Euro", Symbol: "E", Decimals: 2, IsBase: true, Rate: one}, b.now); err != nil {
		t.Fatal(err)
	}
	if b.rsd, err = b.ledger.InsertCurrency(ctx,
		ledgerrepo.Currency{Code: "RSD", Name: "Dinar", Symbol: "d", Decimals: 2, Rate: one}, b.now); err != nil {
		t.Fatal(err)
	}
	account := func(name string, cur int64) int64 {
		id, err := b.ledger.InsertAccount(ctx, ledgerrepo.AccountFields{
			Name: name, Type: "bank", CurrencyID: cur, IsActive: true,
			InitialBalance: money.New(0, money.Unit{ID: cur, Decimals: 2}),
		}, b.now)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	b.card, b.dinar = account("Card", b.eur), account("Dinar account", b.rsd)
	// The record comes first: a space payment's uuid is a foreign key to it.
	rec := transfersrepo.Record{UUID: "t1", Payload: ptr(payload), SigV: 1, Status: "confirmed"}
	if err := b.repo.Save(ctx, rec, b.now); err != nil {
		t.Fatal(err)
	}
	return b
}

// mirror is the transaction whose space payment carries uuid.
func (b *books) mirror(t *testing.T, uuid string) *ledgerrepo.Tx {
	t.Helper()
	list, _, err := b.ledger.Filtered(ctx, filter.TxFilter{}, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range list {
		for _, p := range x.Payments {
			if p.SpaceTransferUUID != nil && *p.SpaceTransferUUID == uuid {
				return &x
			}
		}
	}
	return nil
}

func TestMirrorIsATransactionWithARealAndASpacePayment(t *testing.T) {
	b := newBooks(t)
	m := transfersrepo.Mirror{
		UUID: "t1", Type: "transfer_out", Date: "2026-03-04", Status: "confirmed", AccountID: b.card, Amount: -1000,
		Description: ptr("rent"),
	}
	if has, _ := b.repo.HasMirror(ctx, "t1"); has {
		t.Fatal("a mirror before it was saved")
	}
	if err := b.repo.SaveMirror(ctx, m, b.now); err != nil {
		t.Fatal(err)
	}
	tx := b.mirror(t, "t1")
	if tx == nil || tx.Type != "transfer_out" || tx.Status != "confirmed" || len(tx.Payments) != 2 || len(tx.Lines) != 0 {
		t.Fatalf("mirror = %+v", tx)
	}
	space, _ := b.ledger.SpaceAccountID(ctx, b.eur)
	if space == nil {
		t.Fatal("no space account was made for the currency")
	}
	for _, p := range tx.Payments {
		switch p.AccountID {
		case b.card:
			if p.Amount.Minor() != -1000 || p.SpaceTransferUUID != nil {
				t.Fatalf("real payment = %+v", p)
			}
		case *space:
			if p.Amount.Minor() != 1000 || p.SpaceTransferUUID == nil {
				t.Fatalf("space payment = %+v", p)
			}
		default:
			t.Fatalf("payment on account %d", p.AccountID)
		}
	}

	// Saving again updates in place: the same transaction, no second space account.
	m.Amount, m.Status = -2500, "pending"
	if err := b.repo.SaveMirror(ctx, m, b.now); err != nil {
		t.Fatal(err)
	}
	again := b.mirror(t, "t1")
	if again == nil || again.ID != tx.ID || again.Status != "pending" || len(again.Payments) != 2 {
		t.Fatalf("mirror after saving again = %+v", again)
	}
	if n := b.ledger.CountTransactions(ctx); n != 1 {
		t.Fatalf("transactions = %d, want the same one", n)
	}
	for _, p := range again.Payments {
		if p.AccountID == b.card && p.Amount.Minor() != -2500 {
			t.Fatalf("real payment after saving again = %+v", p)
		}
	}
}

func TestMirrorFollowsTheAccountToAnotherCurrency(t *testing.T) {
	b := newBooks(t)
	m := transfersrepo.Mirror{
		UUID: "t1", Type: "transfer_in", Date: "2026-03-04", Status: "confirmed", AccountID: b.card, Amount: 1000,
	}
	if err := b.repo.SaveMirror(ctx, m, b.now); err != nil {
		t.Fatal(err)
	}
	m.AccountID, m.Amount = b.dinar, 117000
	if err := b.repo.SaveMirror(ctx, m, b.now); err != nil {
		t.Fatal(err)
	}
	tx := b.mirror(t, "t1")
	rsdSpace, _ := b.ledger.SpaceAccountID(ctx, b.rsd)
	if tx == nil || len(tx.Payments) != 2 || rsdSpace == nil {
		t.Fatalf("mirror = %+v", tx)
	}
	for _, p := range tx.Payments {
		if p.AccountID != b.dinar && p.AccountID != *rsdSpace {
			t.Fatalf("a payment is left on account %d, of the old currency", p.AccountID)
		}
		if p.AccountID == *rsdSpace && (p.Amount.Minor() != -117000 || p.SpaceTransferUUID == nil) {
			t.Fatalf("space payment = %+v", p)
		}
	}
}

func TestDeletingARecordDeletesItsMirror(t *testing.T) {
	b := newBooks(t)
	if got, err := b.repo.AccountCurrency(ctx, 999); got != nil || err != nil {
		t.Fatalf("unknown account = %v, %v", got, err)
	}
	if got, _ := b.repo.AccountCurrency(ctx, b.card); got == nil || got.Code != "EUR" || got.Decimals != 2 || got.IsService {
		t.Fatalf("account currency = %+v", got)
	}

	m := transfersrepo.Mirror{
		UUID: "t1", Type: "transfer_out", Date: "2026-03-04", Status: "confirmed", AccountID: b.card, Amount: -1000,
	}
	if err := b.repo.SaveMirror(ctx, m, b.now); err != nil {
		t.Fatal(err)
	}
	space, _ := b.ledger.SpaceAccountID(ctx, b.eur)
	if got, _ := b.repo.AccountCurrency(ctx, *space); got == nil || !got.IsService {
		t.Fatalf("the space account is not a service account: %+v", got)
	}

	if err := b.repo.Delete(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	if rec, _ := b.repo.Record(ctx, "t1"); rec != nil {
		t.Fatal("the record survived")
	}
	if has, _ := b.repo.HasMirror(ctx, "t1"); has {
		t.Fatal("the mirror survived")
	}
	if n := b.ledger.CountTransactions(ctx); n != 0 {
		t.Fatalf("transactions = %d", n)
	}
	// Deleting what is not there is not an error.
	if err := b.repo.Delete(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
}
