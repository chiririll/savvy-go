package domain

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/migrate"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

type transferEnv struct {
	st     *sqlite.Store
	tr     Transfers
	owner  *auth.User
	A, B   *Space
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
	spaces := Spaces{Store: st}
	pass := "secret1"
	owner, err := (auth.Users{DB: st.Server()}).Create(ctx, "Owner", "o@test.com", &pass, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	A, _ := spaces.Create(ctx, "A", owner)
	B, _ := spaces.Create(ctx, "B", owner)
	e := &transferEnv{st: st, owner: owner, A: A, B: B, crash: &crash, dir: dir, holder: holder,
		tr: Transfers{Spaces: spaces, Keys: KeyRing{Holder: holder, Trusted: TrustedKeys(st.Server())}}}
	e.accA, e.accB = e.account(t, A.ID, "EUR"), e.account(t, B.ID, "USD")
	if err := e.tr.Link(ctx, owner, A.ID, B.ID); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e *transferEnv) account(t *testing.T, spaceID int64, code string) int64 {
	t.Helper()
	d, _ := e.st.Space(context.Background(), spaceID)
	cur, err := (Currencies{DB: d}).Create(context.Background(), Currency{Code: code, Name: code, Symbol: code, Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := (Accounts{DB: d}).Create(context.Background(), AccountInput{Name: code, Type: "cash", CurrencyID: cur.ID})
	if err != nil {
		t.Fatal(err)
	}
	return acc.ID
}

func (e *transferEnv) send(t *testing.T, amount string) *Transfer {
	t.Helper()
	tr, err := e.tr.Create(context.Background(), e.owner, TransferInput{
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
func TestP22CrashBetweenCommitsIsMerged(t *testing.T) {
	e := newTransferEnv(t)
	*e.crash = true
	_, err := e.tr.Create(context.Background(), e.owner, TransferInput{
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
	e.tr.Keys.sign(&newer)
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
	e.tr.Keys.sign(&gone)
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
	e.tr.Keys.sign(&x)
	e.tr.Keys.sign(&y)
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
func TestP43UnsignedRecordIsNotMerged(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "10")
	dB, _ := e.st.Space(ctx, e.B.ID)
	forged := *tr
	forged.Version, forged.To.Amount = 9, 999999
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
	if tr.SigV != 1 || !e.tr.Keys.verify(*tr) {
		t.Fatal("a fresh record does not verify")
	}
	future := *tr
	future.SigV = 99
	if e.tr.Keys.verify(future) {
		t.Fatal("an unknown format verified")
	}
}

// P47: after a key rotation old records still verify and new ones are
// signed with the new key.
func TestP47KeyRotation(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	old := e.send(t, "1")
	oldKID := e.holder.Key().KID()
	if _, err := e.tr.RotateKey(ctx, e.owner, e.dir); err != nil {
		t.Fatal(err)
	}
	if e.holder.Key().KID() == oldKID {
		t.Fatal("the key did not change")
	}
	if !e.tr.Keys.verify(e.records(t, e.A.ID)[old.UUID]) {
		t.Fatal("a record signed with the old key no longer verifies")
	}
	if fresh := e.send(t, "2"); fresh.SignerKID != e.holder.Key().KID() {
		t.Fatal("new records are not signed with the new key")
	}
}

// P27, P46: spaces moved together to a server with another key keep their
// uuids; their transfers do not verify there until the old key is trusted
// or the members re-sign them, and then they merge again.
func TestP27P46MoveToAnotherServer(t *testing.T) {
	e := newTransferEnv(t)
	ctx := context.Background()
	tr := e.send(t, "4")
	exportA, exportB := filepath.Join(t.TempDir(), "a.sqlite"), filepath.Join(t.TempDir(), "b.sqlite")
	if err := e.st.ExportSpace(ctx, e.A.ID, exportA); err != nil {
		t.Fatal(err)
	}
	if err := e.st.ExportSpace(ctx, e.B.ID, exportB); err != nil {
		t.Fatal(err)
	}

	n := newTransferEnv(t) // another server, another key
	importSpace := func(path string) *Space {
		p, err := n.st.PrepareSpace(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		sp, err := n.tr.Spaces.createFrom(ctx, "moved", n.owner, p)
		if err != nil {
			t.Fatal(err)
		}
		return sp
	}
	a2, b2 := importSpace(exportA), importSpace(exportB)
	if a2.UUID != e.A.UUID || b2.UUID != e.B.UUID {
		t.Fatal("moved spaces lost their uuids")
	}
	if n.tr.Keys.verify(n.records(t, a2.ID)[tr.UUID]) {
		t.Fatal("a foreign signature verified")
	}
	if err := n.tr.Link(ctx, n.owner, a2.ID, b2.ID); err != nil {
		t.Fatal(err)
	}
	// An update on the old server's record is refused until it is trusted.
	if _, err := n.tr.Update(ctx, n.owner, a2.ID, tr.UUID, TransferUpdate{Date: ptr("2026-10-02")}); !errors.Is(err, ErrUnverified) {
		t.Fatalf("update of an unverified record: %v", err)
	}
	pub := base64.StdEncoding.EncodeToString(e.holder.Key().Public())
	if _, err := n.tr.TrustKey(ctx, n.owner, "old server", pub); err != nil {
		t.Fatal(err)
	}
	if !n.tr.Keys.verify(n.records(t, a2.ID)[tr.UUID]) {
		t.Fatal("trusting the old key did not verify the record")
	}
	updated, err := n.tr.Update(ctx, n.owner, a2.ID, tr.UUID, TransferUpdate{Date: ptr("2026-10-02")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.SignerKID != n.holder.Key().KID() || n.records(t, b2.ID)[tr.UUID].Date != "2026-10-02" {
		t.Fatal("the update was not re-signed here or did not reach the other space")
	}

	// Without trusting the key, members can re-sign a record themselves.
	m := newTransferEnv(t)
	a3 := func() *Space {
		p, _ := m.st.PrepareSpace(ctx, exportA)
		sp, _ := m.tr.Spaces.createFrom(ctx, "moved", m.owner, p)
		return sp
	}()
	if err := m.tr.Trust(ctx, m.owner, a3.ID, tr.UUID); err != nil {
		t.Fatal(err)
	}
	if !m.tr.Keys.verify(m.records(t, a3.ID)[tr.UUID]) {
		t.Fatal("re-signing did not make the record verify")
	}
}

var _ store.Store = (*sqlite.Store)(nil)
