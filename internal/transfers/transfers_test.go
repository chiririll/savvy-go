package transfers

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	spacekeysrepo "savvy-go/internal/db/repo/spacekeys"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money"
	"savvy-go/internal/signing"
	"savvy-go/internal/spacekeys"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

type transferEnv struct {
	st     *sqlite.Store
	tr     Service
	owner  *auth.User
	A, B   *spaces.Space
	accA   int64
	accB   int64
	crash  *bool
	dir    string
	holder *signing.Holder
}

func newTransferEnv(t *testing.T) *transferEnv {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	crash := false
	st, err := sqlite.Open(ctx, sqlite.Options{
		Dir: dir, MigrateServer: migrate.Server.Up, MigrateSpace: migrate.Space.Up,
		BetweenCommits: func(int) error {
			if crash {
				return errors.New("crash")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	key, err := signing.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	holder := signing.NewHolder(key)
	sps := spaces.Service{Store: st}
	pass := "secret1"
	owner, err := (auth.Users{DB: st.Server()}).Create(ctx, "Owner", "o@test.com", &pass, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	A, _ := sps.Create(ctx, "A", owner)
	B, _ := sps.Create(ctx, "B", owner)
	e := &transferEnv{st: st, owner: owner, A: A, B: B, crash: &crash, dir: dir, holder: holder,
		tr: Service{Spaces: sps, Keys: signing.KeyRing{Holder: holder, Trusted: TrustedKeys(st.Server())}}}
	e.accA, e.accB = e.account(t, A.ID, "EUR"), e.account(t, B.ID, "USD")
	if err := e.tr.Link(ctx, owner, A.ID, B.ID); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e *transferEnv) account(t *testing.T, spaceID int64, code string) int64 {
	t.Helper()
	ctx := context.Background()
	d, _ := e.st.Space(ctx, spaceID)
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	cur, err := l.InsertCurrency(
		ctx, ledgerrepo.Currency{Code: code, Name: code, Symbol: code, Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)}, now)
	if err != nil {
		t.Fatal(err)
	}
	acc, err := l.InsertAccount(ctx, ledgerrepo.AccountFields{
		Name: code, Type: "cash", CurrencyID: cur, IsActive: true, InitialBalance: money.New(0, money.Unit{ID: cur, Decimals: 2}),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	return acc
}

func (e *transferEnv) send(t *testing.T, amount string) *Transfer {
	t.Helper()
	tr, err := e.tr.Create(context.Background(), e.owner, Input{
		FromSpace: e.A.ID, FromAccount: e.accA, FromAmount: decimal.RequireFromString(amount),
		ToSpace: e.B.ID, ToAccount: e.accB, ToAmount: decimal.RequireFromString(amount), Date: "2026-10-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func (e *transferEnv) records(t *testing.T, spaceID int64) map[string]Transfer {
	t.Helper()
	d, _ := e.st.Space(context.Background(), spaceID)
	all, err := loadTransfers(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// P22: a crash between the two commits leaves the transfer in A only; the
// merge at startup finishes it in B.
func TestP22CrashBetweenCommitsMerged(t *testing.T) {
	e := newTransferEnv(t)
	*e.crash = true
	_, err := e.tr.Create(context.Background(), e.owner, Input{
		FromSpace: e.A.ID, FromAccount: e.accA, FromAmount: decimal.NewFromInt(5),
		ToSpace: e.B.ID, ToAccount: e.accB, ToAmount: decimal.NewFromInt(5), Date: "2026-10-01",
	})
	if err == nil {
		t.Fatal("want the simulated crash")
	}
	*e.crash = false
	if len(e.records(t, e.A.ID)) != 1 || len(e.records(t, e.B.ID)) != 0 {
		t.Fatal("want the transfer in A only")
	}
	if err := e.tr.SyncAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(e.records(t, e.B.ID)) != 1 {
		t.Fatal("the merge did not finish the transfer in B")
	}
}

// P23: the merge takes the higher version, a newer tombstone wins, equal
// versions with different content resolve the same way on both sides, and a
// second run changes nothing.
func TestP23MergeRules(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dA, _ := e.st.Space(ctx, e.A.ID)
	dB, _ := e.st.Space(ctx, e.B.ID)

	// B is ahead: a newer version written only there.
	newer := *tr
	newer.Version, newer.From.Amount = 2, 2000
	e.sign(t, &newer)
	if err := writeRecord(ctx, dB, e.B.UUID, newer, "confirmed", nil, nil); err != nil {
		t.Fatal(err)
	}
	_ = e.tr.SyncAll(ctx)
	if got := e.records(t, e.A.ID)[tr.UUID]; got.Version != 2 || got.From.Amount != 2000 {
		t.Fatalf("A did not take the newer version: %+v", got)
	}

	// A tombstone with a higher version removes it in B.
	gone := newer
	gone.Version = 3
	at := now()
	gone.DeletedAt = &at
	e.sign(t, &gone)
	if err := writeRecord(ctx, dA, e.A.UUID, gone, "confirmed", nil, nil); err != nil {
		t.Fatal(err)
	}
	_ = e.tr.SyncAll(ctx)
	if got := e.records(t, e.B.ID)[tr.UUID]; got.DeletedAt == nil {
		t.Fatal("the tombstone did not win")
	}

	// Same version, different content: both sides settle on the same one.
	other := e.send(t, "1")
	x, y := *other, *other
	x.Version, y.Version = 5, 5
	x.Description, y.Description = ptr("x"), ptr("y")
	e.sign(t, &x)
	e.sign(t, &y)
	_ = writeRecord(ctx, dA, e.A.UUID, x, "confirmed", nil, nil)
	_ = writeRecord(ctx, dB, e.B.UUID, y, "confirmed", nil, nil)
	_ = e.tr.SyncAll(ctx)
	a, b := e.records(t, e.A.ID)[other.UUID], e.records(t, e.B.ID)[other.UUID]
	if !a.same(b) || !a.same(winner(x, y)) {
		t.Fatalf("sides disagree: %v / %v", *a.Description, *b.Description)
	}

	before := e.records(t, e.B.ID)
	_ = e.tr.SyncAll(ctx)
	after := e.records(t, e.B.ID)
	for id, r := range before {
		if !r.same(after[id]) || r.Version != after[id].Version {
			t.Fatal("a second merge changed something")
		}
	}
}

// P43: the merge never copies a record whose signature does not verify.
func TestP43UnsignedRecordNotMerged(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dB, _ := e.st.Space(ctx, e.B.ID)
	forged := *tr
	forged.Version, forged.To.Amount = 9, 999999
	forged.signed = nil // the edit is made on the fields, and the old signature stays
	if err := writeRecord(ctx, dB, e.B.UUID, forged, "confirmed", nil, nil); err != nil {
		t.Fatal(err)
	}
	_ = e.tr.SyncAll(ctx)
	if got := e.records(t, e.A.ID)[tr.UUID]; got.Version != 1 {
		t.Fatalf("an unsigned version reached A: %+v", got)
	}
}

// P49: a record verifies by the signature format it was made with; an
// unknown format does not verify.
func TestP49SignatureFormatVersion(t *testing.T) {
	e := newTransferEnv(t)
	tr := e.send(t, "1")
	if tr.SigV != 2 || !e.ok(*tr) {
		t.Fatal("a fresh record does not verify")
	}
	future := *tr
	future.SigV = 99
	if e.ok(future) {
		t.Fatal("an unknown format verified")
	}
}

// A transfer is signed by the key of each of its spaces, not by the server.
func TestTransferIsSignedByBothSpaceKeys(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "1")
	keyOf := func(id int64) string {
		d, _ := e.st.Space(ctx, id)
		k, err := spacekeys.Current(ctx, d)
		if err != nil {
			t.Fatal(err)
		}
		return k.KID()
	}
	if tr.SignerKID != keyOf(e.A.ID) || tr.DestKID != keyOf(e.B.ID) || tr.SignerKID == e.holder.Key().KID() {
		t.Fatalf("signed by %s and %s, want the keys of A (%s) and B (%s)",
			tr.SignerKID, tr.DestKID, keyOf(e.A.ID), keyOf(e.B.ID))
	}
	for _, id := range []int64{e.A.ID, e.B.ID} {
		got := e.records(t, id)[tr.UUID]
		if got.DestKID != tr.DestKID || got.DestSignature != tr.DestSignature || !e.ok(got) {
			t.Fatalf("space %d stores %+v", id, got)
		}
	}

	// The server key no longer matters to a transfer.
	if _, err := e.tr.RotateKey(ctx, e.owner, e.dir); err != nil {
		t.Fatal(err)
	}
	if !e.ok(e.records(t, e.A.ID)[tr.UUID]) {
		t.Fatal("rotating the server key broke a transfer")
	}
}

// One missing signature, or the two swapped, fails the check.
func TestOneSignatureIsNotEnough(t *testing.T) {
	e := newTransferEnv(t)
	tr := e.send(t, "1")
	noDest := *tr
	noDest.DestKID, noDest.DestSignature = "", ""
	swapped := *tr
	swapped.SignerKID, swapped.DestKID = tr.DestKID, tr.SignerKID
	swapped.Signature, swapped.DestSignature = tr.DestSignature, tr.Signature
	for name, x := range map[string]Transfer{"no destination": noDest, "swapped": swapped} {
		if e.ok(x) {
			t.Fatalf("%s: verified", name)
		}
	}
}

// A retired space key still verifies what it signed; a new transfer uses the
// new key.
func TestSpaceKeyRotation(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	old := e.send(t, "1")
	d, _ := e.st.Space(ctx, e.A.ID)
	var fresh *signing.Key
	if err := store.Tx(ctx, d, func(tx store.DB) (err error) {
		fresh, err = spacekeys.Rotate(ctx, tx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fresh.KID() == old.SignerKID {
		t.Fatal("the key did not change")
	}
	if !e.ok(e.records(t, e.A.ID)[old.UUID]) {
		t.Fatal("a record signed with the retired key no longer verifies")
	}
	if next := e.send(t, "2"); next.SignerKID != fresh.KID() || !e.ok(*next) {
		t.Fatal("a new record is not signed with the new key")
	}
}

// Spaces made here have a key from the start.
func TestSpaceHasAKeyFromCreation(t *testing.T) {
	e := newTransferEnv(t)
	d, _ := e.st.Space(context.Background(), e.A.ID)
	keys, err := (spacekeysrepo.Repo{DB: d}).List(context.Background())
	if err != nil || len(keys) != 1 || keys[0].RetiredAt != nil {
		t.Fatalf("keys = %+v, %v", keys, err)
	}
}

// A space without a key (a converted v1.0 file, an old backup) gets one at
// startup, and a space that has one keeps it.
func TestEnsureKeys(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	dA, _ := e.st.Space(ctx, e.A.ID)
	dB, _ := e.st.Space(ctx, e.B.ID)
	before, _ := (spacekeysrepo.Repo{DB: dB}).List(ctx)
	if _, err := dA.ExecContext(ctx, `DELETE FROM space_keys`); err != nil {
		t.Fatal(err)
	}
	if err := e.tr.EnsureKeys(ctx); err != nil {
		t.Fatal(err)
	}
	a, _ := (spacekeysrepo.Repo{DB: dA}).List(ctx)
	b, _ := (spacekeysrepo.Repo{DB: dB}).List(ctx)
	if len(a) != 1 || len(b) != 1 || b[0].KID != before[0].KID {
		t.Fatalf("keys after = %+v / %+v", a, b)
	}
}

// signV1 signs as v1.0 did: the server key over the payload, nothing else.
func (e *transferEnv) signV1(x *Transfer) {
	key := e.holder.Key()
	x.SigV, x.signed = 1, nil
	p, _ := x.payload(1)
	x.signed = p
	x.SignerKID, x.Signature = key.KID(), base64.StdEncoding.EncodeToString(key.Sign(p))
	x.DestKID, x.DestSignature = "", ""
}

// writeV1 puts a v1.0 record into both spaces.
func (e *transferEnv) writeV1(t *testing.T, x Transfer) {
	t.Helper()
	ctx := context.Background()
	for _, sp := range []*spaces.Space{e.A, e.B} {
		d, _ := e.st.Space(ctx, sp.ID)
		if err := writeRecord(ctx, d, sp.UUID, x, transferConfirmed, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

// P46: transfers v1.0 left signed by the server are re-signed by the space
// keys over the very same bytes, and only when the server signature holds and
// the two spaces agree.
func TestResignV1Transfers(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	good, bad, apart := *e.send(t, "1"), *e.send(t, "2"), *e.send(t, "3")
	for _, x := range []*Transfer{&good, &bad, &apart} {
		e.signV1(x)
	}
	bad.Signature = base64.StdEncoding.EncodeToString([]byte("not a signature"))
	e.writeV1(t, good)
	e.writeV1(t, bad)
	e.writeV1(t, apart)
	dB, _ := e.st.Space(ctx, e.B.ID)
	later := apart
	later.Version = 2
	e.signV1(&later)
	if err := writeRecord(ctx, dB, e.B.UUID, later, transferConfirmed, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := e.tr.Resign(ctx); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{e.A.ID, e.B.ID} {
		got := e.records(t, id)
		if g := got[good.UUID]; g.SigV != 2 || !e.ok(g) || string(g.signed) != string(good.signed) {
			t.Fatalf("space %d: re-signed record = %+v", id, g)
		}
		if got[bad.UUID].SigV != 1 {
			t.Fatalf("space %d: a record with a bad server signature was re-signed", id)
		}
		if got[apart.UUID].SigV != 1 {
			t.Fatalf("space %d: a record the spaces disagree on was re-signed", id)
		}
	}
	if e.mirror(t, e.A.ID, good.UUID) == nil {
		t.Fatal("re-signing lost the transaction of the transfer")
	}

	// A second run changes nothing.
	if err := e.tr.Resign(ctx); err != nil {
		t.Fatal(err)
	}
}

// moveSpaces exports the spaces of e and imports them on a new server; with
// onlyA the second one stays behind.
func (e *transferEnv) moveSpaces(t *testing.T, onlyA bool) (*transferEnv, *spaces.Space, *spaces.Space) {
	t.Helper()
	ctx := context.Background()
	n := newTransferEnv(t) // another server, another key
	importSpace := func(id int64) *spaces.Space {
		path := filepath.Join(t.TempDir(), "space.sqlite")
		if err := e.st.ExportSpace(ctx, id, path); err != nil {
			t.Fatal(err)
		}
		p, err := n.st.PrepareSpace(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		sp, err := n.tr.Spaces.CreateFrom(ctx, "moved", n.owner, p)
		if err != nil {
			t.Fatal(err)
		}
		return sp
	}
	a2 := importSpace(e.A.ID)
	if onlyA {
		return n, a2, nil
	}
	return n, a2, importSpace(e.B.ID)
}

// P27: spaces moved together to another server keep their transfers, signed
// and editable; the keys travelled with the spaces.
func TestP27MovedTogetherKeepTransfers(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "4")
	n, a2, b2 := e.moveSpaces(t, false)
	if a2.UUID != e.A.UUID || b2.UUID != e.B.UUID {
		t.Fatal("moved spaces lost their uuids")
	}
	if !n.ok(n.records(t, a2.ID)[tr.UUID]) || !n.ok(n.records(t, b2.ID)[tr.UUID]) {
		t.Fatal("the transfer does not verify on the new server")
	}
	if err := n.tr.Link(ctx, n.owner, a2.ID, b2.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := n.tr.Update(ctx, n.owner, a2.ID, tr.UUID, Update{Date: ptr("2026-10-02")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.SignerKID != tr.SignerKID || updated.DestKID != tr.DestKID ||
		n.records(t, b2.ID)[tr.UUID].Date != "2026-10-02" {
		t.Fatal("the update was not signed with the travelled keys or did not reach the other space")
	}
}

// P46: one moved space alone cannot vouch for a transfer: it is frozen until
// the other space is here too.
func TestP46OneMovedSpaceIsFrozen(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "4")
	n, a2, _ := e.moveSpaces(t, true)
	if n.ok(n.records(t, a2.ID)[tr.UUID]) {
		t.Fatal("a transfer verified without its other space")
	}
	_, err := n.tr.Update(ctx, n.owner, a2.ID, tr.UUID, Update{Date: ptr("2026-10-02")})
	if !errors.Is(err, ErrUnverified) {
		t.Fatalf("update of a frozen transfer: %v", err)
	}
	views, err := n.tr.List(ctx, a2.ID)
	if err != nil || len(views) != 1 || views[0].Verified || !views[0].Frozen {
		t.Fatalf("view = %+v, %v", views, err)
	}
}

// A backup of one space with an edited transfer cannot be made valid by that
// space alone: the other space's key signs too.
func TestForgedTransferFailsTheOtherSpacesKey(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	forged := *tr
	forged.From.Amount, forged.To.Amount = 999999, 999999
	forged.signed = nil
	// A's admin signs with a key of their own, put in A's file.
	dA, _ := e.st.Space(ctx, e.A.ID)
	var mine *signing.Key
	if err := store.Tx(ctx, dA, func(tx store.DB) (err error) {
		mine, err = spacekeys.Rotate(ctx, tx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	p, _ := forged.payload(2)
	forged.signed = p
	forged.SignerKID, forged.Signature = mine.KID(), base64.StdEncoding.EncodeToString(mine.Sign(p))
	if e.ok(forged) {
		t.Fatal("a transfer signed by one space verified")
	}
}

// Trust re-signs a transfer the keys of its spaces no longer vouch for, after
// the members of both agree.
func TestTrustResignsWithCurrentKeys(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "4")
	dA, _ := e.st.Space(ctx, e.A.ID)
	if _, err := dA.ExecContext(ctx, `DELETE FROM space_keys`); err != nil { // A's key is gone
		t.Fatal(err)
	}
	if e.ok(e.records(t, e.A.ID)[tr.UUID]) {
		t.Fatal("verified without A's key")
	}
	if err := e.tr.Trust(ctx, e.owner, e.A.ID, tr.UUID); err != nil {
		t.Fatal(err)
	}
	if !e.ok(e.records(t, e.A.ID)[tr.UUID]) || !e.ok(e.records(t, e.B.ID)[tr.UUID]) {
		t.Fatal("re-signing did not make the record verify")
	}
}

// sign signs x with the current keys of the env's two spaces.
func (e *transferEnv) sign(t *testing.T, x *Transfer) {
	t.Helper()
	ctx := context.Background()
	err := e.st.InSpaces(ctx, []int64{e.A.ID, e.B.ID}, func(dbs map[int64]store.DB) error {
		return e.tr.signWithSpaces(ctx, dbs, x)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ok reports whether x verifies on this env's server.
func (e *transferEnv) ok(x Transfer) bool { return e.tr.verify(context.Background(), x) }

var _ store.Store = (*sqlite.Store)(nil)

// An update keeps a description it does not set; HasDescription with nil
// clears it on both sides.
func TestTransferUpdateClearsDescriptionOnRequest(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr, err := e.tr.Create(ctx, e.owner, Input{
		FromSpace: e.A.ID, FromAccount: e.accA, FromAmount: decimal.NewFromInt(10),
		ToSpace: e.B.ID, ToAccount: e.accB, ToAmount: decimal.NewFromInt(10), Date: "2026-10-01",
		Description: ptr("Rent share"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.tr.Update(ctx, e.owner, e.A.ID, tr.UUID, Update{Date: ptr("2026-10-02")}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{e.A.ID, e.B.ID} {
		if d := e.records(t, id)[tr.UUID].Description; d == nil || *d != "Rent share" {
			t.Fatalf("space %d: description %v, want kept", id, d)
		}
	}

	if _, err := e.tr.Update(ctx, e.owner, e.A.ID, tr.UUID, Update{HasDescription: true}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{e.A.ID, e.B.ID} {
		if d := e.records(t, id)[tr.UUID].Description; d != nil {
			t.Fatalf("space %d: description %q, want cleared", id, *d)
		}
	}
}
