package transfers

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/money"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// A transfer between two linked spaces is one record kept in both spaces'
// databases (space_transfers), plus in each a transaction for its own side
// (transfer_out / transfer_in) that balances and reports use. Writes go to
// both spaces in one store.InSpaces. Records carry a version and the server's
// signature; a merge by version keeps the two copies in step and a deletion is
// a tombstone with a version of its own.
//
// Two rules keep one space from changing another without its members (the
// sync invariant): the automatic merge only finishes a write that InSpaces
// started (it never moves a record under review or one whose signature this
// server does not trust), and after a space is restored every difference
// becomes a pending record in that space alone until someone decides.

// Side is one space's side of a transfer. AccountID belongs to that
// space; the other space only stores it as an opaque number.
type Side struct {
	SpaceUUID string `json:"space"`
	AccountID int64  `json:"account"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Decimals  int64  `json:"decimals"`
}

// Transfer is a transfer record.
type Transfer struct {
	UUID        string  `json:"uuid"`
	From        Side    `json:"from"`
	To          Side    `json:"to"`
	Date        string  `json:"date"`
	Description *string `json:"description"`
	CreatedBy   *int64  `json:"created_by"`
	Version     int64   `json:"version"`
	DeletedAt   *string `json:"deleted_at"`
	SigV        int64   `json:"sig_v"`
	SignerKID   string  `json:"signer_kid"`
	Signature   string  `json:"signature"`
	// Local to one space, never signed or copied.
	Status string    `json:"-"`
	Review *string   `json:"-"`
	Remote *Transfer `json:"-"`
}

const (
	transferOK             = "ok"
	transferNeedsAttention = "needs_attention"
	transferPending        = "pending"

	reviewCreatedRemote = "created_remote"
	reviewDeletedRemote = "deleted_remote"
	reviewChangedRemote = "changed_remote"
	reviewMissingRemote = "missing_remote"

	// transferReviewKey marks a restored space whose transfers wait for a
	// review merge; it is written before the restored file goes live.
	transferReviewKey = repo.ReviewKey
)

// MarkTransferReview flags a prepared space so its transfers are reviewed
// with their linked spaces once it goes live, and not merged automatically.
// The mark goes in before the file does: if the process dies before the
// review, startup reviews again (P24).
func MarkTransferReview(p *store.PreparedSpace) {
	if p.Settings == nil {
		p.Settings = map[string]string{}
	}
	p.Settings[transferReviewKey] = now()
}

var (
	ErrNotLinked      = errors.New("these spaces are not linked")
	ErrSameSpace      = errors.New("a transfer between spaces needs two different spaces")
	ErrRights         = errors.New("you need to be an editor in both spaces")
	ErrAccountMissing = errors.New("the account does not exist in that space")
	ErrUnverified     = errors.New("this transfer's signature cannot be verified on this server")
	ErrNotUnderReview = errors.New("this transfer is not waiting for a decision")
	ErrPending        = errors.New("decide on this transfer's pending change first")
	ErrGone           = errors.New("transfer not found")
)

// --- signing ---------------------------------------------------------------

// transferPayloadV1 is what sig_v 1 signs. A later format adds fields in a
// new struct and a new sig_v; records keep verifying by the format they were
// signed with (P49).
type transferPayloadV1 struct {
	UUID        string  `json:"uuid"`
	From        Side    `json:"from"`
	To          Side    `json:"to"`
	Date        string  `json:"date"`
	Description *string `json:"description"`
	CreatedBy   *int64  `json:"created_by"`
	Version     int64   `json:"version"`
	DeletedAt   *string `json:"deleted_at"`
}

const currentSigV = 1

func (t Transfer) payload(sigV int64) ([]byte, bool) {
	switch sigV {
	case 1:
		b, err := json.Marshal(transferPayloadV1{t.UUID, t.From, t.To, t.Date, t.Description, t.CreatedBy, t.Version, t.DeletedAt})
		return b, err == nil
	}
	return nil, false
}

func signTransfer(k signing.KeyRing, t *Transfer) {
	key := k.Holder.Key()
	t.SigV = currentSigV
	p, _ := t.payload(currentSigV)
	t.SignerKID = key.KID()
	t.Signature = base64.StdEncoding.EncodeToString(key.Sign(p))
}

// verify reports whether t is signed by this server or a key it trusts.
func verifyTransfer(k signing.KeyRing, t Transfer) bool {
	p, ok := t.payload(t.SigV)
	if !ok {
		return false
	}
	return k.Check(t.SignerKID, p, []byte(t.Signature)) != signing.Unsigned
}

// TrustedKeys looks up a trusted public key in the server database.
func TrustedKeys(server store.DB) func(kid string) ed25519.PublicKey {
	return func(kid string) ed25519.PublicKey {
		raw, ok := (repo.Server{DB: server}).PublicKey(context.Background(), kid)
		if !ok {
			return nil
		}
		pub, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			return nil
		}
		return pub
	}
}

// --- rows ------------------------------------------------------------------

func sideFromRecord(r repo.Side) Side { return Side(r) }

func transferFromRecord(r repo.Record) Transfer {
	t := Transfer{
		UUID: r.UUID, From: sideFromRecord(r.From), To: sideFromRecord(r.To),
		Date: r.Date, Description: r.Description, CreatedBy: r.CreatedBy, Version: r.Version, DeletedAt: r.DeletedAt,
		SigV: r.SigV, SignerKID: r.SignerKID, Signature: r.Signature, Status: r.Status, Review: r.Review,
	}
	if r.Remote != "" {
		var remote Transfer
		if json.Unmarshal([]byte(r.Remote), &remote) == nil {
			t.Remote = &remote
		}
	}
	return t
}

func recordOf(t Transfer, status string, review *string, remote *Transfer) repo.Record {
	rec := repo.Record{
		UUID: t.UUID, From: repo.Side(t.From), To: repo.Side(t.To),
		Date: t.Date, Description: t.Description, CreatedBy: t.CreatedBy, Version: t.Version, DeletedAt: t.DeletedAt,
		SigV: t.SigV, SignerKID: t.SignerKID, Signature: t.Signature, Status: status, Review: review,
	}
	if remote != nil {
		b, _ := json.Marshal(remote)
		rec.Remote = string(b)
	}
	return rec
}

func loadTransfers(ctx context.Context, d store.DB) (map[string]Transfer, error) {
	rows, err := repo.Space{DB: d}.Records(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Transfer, len(rows))
	for _, r := range rows {
		out[r.UUID] = transferFromRecord(r)
	}
	return out, nil
}

func loadTransfer(ctx context.Context, d store.DB, id string) (*Transfer, error) {
	r, err := repo.Space{DB: d}.Record(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrGone
	}
	t := transferFromRecord(*r)
	return &t, nil
}

// side is own's side of t, and whether own is the sender.
func (t Transfer) side(own string) (Side, bool, bool) {
	switch own {
	case t.From.SpaceUUID:
		return t.From, true, true
	case t.To.SpaceUUID:
		return t.To, false, true
	}
	return Side{}, false, false
}

func (t Transfer) involves(a, b string) bool {
	return (t.From.SpaceUUID == a && t.To.SpaceUUID == b) || (t.From.SpaceUUID == b && t.To.SpaceUUID == a)
}

// same reports whether two records carry the same signed content.
func (t Transfer) same(o Transfer) bool {
	a, _ := t.payload(1)
	b, _ := o.payload(1)
	return bytes.Equal(a, b)
}

// writeRecord stores t in the space whose uuid is own: the record and the
// transaction of own's side, with localStatus ("confirmed" or "pending"). A
// tombstone removes the transaction; an account missing here leaves the
// record needing attention without a transaction. d must be the handle of
// the transaction the caller runs in.
func writeRecord(ctx context.Context, d store.DB, own string, t Transfer, localStatus string, review *string, remote *Transfer) error {
	sp := repo.Space{DB: d}
	side, sender, ok := t.side(own)
	if !ok {
		return fmt.Errorf("transfer %s does not involve this space", t.UUID)
	}
	status := transferOK
	if review != nil {
		status = transferPending
	}
	hasAccount := false
	if t.DeletedAt == nil {
		acc, err := sp.AccountCurrency(ctx, side.AccountID)
		if err != nil {
			return err
		}
		if acc != nil {
			hasAccount = true
		} else if review == nil {
			status = transferNeedsAttention
		}
	}
	at := time.Now().UTC()
	if err := sp.Save(ctx, recordOf(t, status, review, remote), at); err != nil {
		return err
	}
	if t.DeletedAt != nil || !hasAccount {
		return sp.DeleteTransaction(ctx, t.UUID)
	}
	typ := "transfer_in"
	if sender {
		typ = "transfer_out"
	}
	tx := repo.Transaction{
		UUID: t.UUID, Type: typ, Date: t.Date, Status: localStatus, AccountID: side.AccountID, Amount: side.Amount,
		Description: t.Description, CreatedBy: t.CreatedBy,
	}
	exists, err := sp.HasTransaction(ctx, t.UUID)
	if err != nil {
		return err
	}
	if !exists {
		return sp.InsertTransaction(ctx, tx, at)
	}
	return sp.UpdateTransaction(ctx, tx, at)
}

// --- service ---------------------------------------------------------------

// Service links spaces and moves money between them.
type Service struct {
	Spaces spaces.Service
	Keys   signing.KeyRing
}

func (s Service) st() store.Store { return s.Spaces.Store }

func (s Service) server() repo.Server { return repo.Server{DB: s.st().Server()} }

func now() string                       { return time.Now().UTC().Format(time.RFC3339) }
func linkKey(a, b int64) (int64, int64) { return min(a, b), max(a, b) }

// spaceUUIDs maps every registered space's id to its uuid and back.
func (s Service) spaceUUIDs(ctx context.Context) (map[int64]spaces.Space, map[string]spaces.Space, error) {
	rows, err := s.Spaces.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	byID, byUUID := map[int64]spaces.Space{}, map[string]spaces.Space{}
	for _, r := range rows {
		sp := spaces.Space{ID: r.ID, UUID: r.UUID, Name: r.Name}
		byID[r.ID], byUUID[r.UUID] = sp, sp
	}
	return byID, byUUID, nil
}

// Linked reports whether two spaces are linked.
func (s Service) Linked(ctx context.Context, a, b int64) (bool, error) {
	x, y := linkKey(a, b)
	return s.server().Linked(ctx, x, y)
}

// Links lists the spaces linked to a space.
func (s Service) Links(ctx context.Context, spaceID int64) ([]spaces.Space, error) {
	rows, err := s.server().Links(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := []spaces.Space{}
	for _, r := range rows {
		other := r.A
		if other == spaceID {
			other = r.B
		}
		out = append(out, byID[other])
	}
	return out, nil
}

// ErrLinkRights is returned when the linker does not administer both spaces.
var ErrLinkRights = errors.New("you need to be an admin of both spaces to link them")

// Link links two spaces; only who administers both may (P52). Service the
// two already share (spaces moved between servers together) are merged.
func (s Service) Link(ctx context.Context, actor *auth.User, a, b int64) error {
	if a == b {
		return ErrSameSpace
	}
	for _, id := range []int64{a, b} {
		if role, err := s.Spaces.Role(ctx, id, actor.ID); err != nil {
			return err
		} else if role != spaces.RoleAdmin {
			return ErrLinkRights
		}
	}
	x, y := linkKey(a, b)
	if err := s.server().InsertLink(ctx, x, y, actor.ID, time.Now().UTC()); err != nil {
		return err
	}
	return s.sync(ctx, a, b, 0)
}

// Unlink removes a link; an admin of either space may. Service stay in
// both spaces, frozen.
func (s Service) Unlink(ctx context.Context, a, b int64) error {
	x, y := linkKey(a, b)
	ok, err := s.server().DeleteLink(ctx, x, y)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotLinked
	}
	return nil
}

func (s Service) requireEditor(ctx context.Context, actor *auth.User, ids ...int64) error {
	for _, id := range ids {
		role, err := s.Spaces.Role(ctx, id, actor.ID)
		if err != nil {
			return err
		}
		if !spaces.CanWrite(role) {
			return ErrRights
		}
	}
	return nil
}

// Input is a transfer as entered: amounts in each account's currency.
type Input struct {
	FromSpace   int64
	FromAccount int64
	FromAmount  decimal.Decimal
	ToSpace     int64
	ToAccount   int64
	ToAmount    decimal.Decimal
	Date        string
	Description *string
}

// sideFor reads an account's currency in a space and converts amount to it.
func sideFor(ctx context.Context, d store.DB, spaceUUID string, account int64, amount decimal.Decimal) (Side, error) {
	acc, err := repo.Space{DB: d}.AccountCurrency(ctx, account)
	if err != nil {
		return Side{}, err
	}
	if acc == nil {
		return Side{}, ErrAccountMissing
	}
	m, err := money.FromInput(amount, money.Unit{Decimals: int(acc.Decimals)})
	if err != nil {
		return Side{}, err
	}
	if m.Minor() <= 0 {
		return Side{}, errors.New("the amounts must be positive")
	}
	return Side{SpaceUUID: spaceUUID, AccountID: account, Amount: m.Minor(), Currency: acc.Code, Decimals: acc.Decimals}, nil
}

// Create records a transfer in both spaces at once.
func (s Service) Create(ctx context.Context, actor *auth.User, in Input) (*Transfer, error) {
	if in.FromSpace == in.ToSpace {
		return nil, ErrSameSpace
	}
	if err := s.requireEditor(ctx, actor, in.FromSpace, in.ToSpace); err != nil {
		return nil, err
	}
	if ok, err := s.Linked(ctx, in.FromSpace, in.ToSpace); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotLinked
	}
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	id7, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	var t Transfer
	err = s.st().InSpaces(ctx, []int64{in.FromSpace, in.ToSpace}, func(dbs map[int64]store.DB) error {
		from, err := sideFor(ctx, dbs[in.FromSpace], byID[in.FromSpace].UUID, in.FromAccount, in.FromAmount)
		if err != nil {
			return err
		}
		to, err := sideFor(ctx, dbs[in.ToSpace], byID[in.ToSpace].UUID, in.ToAccount, in.ToAmount)
		if err != nil {
			return err
		}
		t = Transfer{UUID: id7.String(), From: from, To: to, Date: in.Date, Description: in.Description, CreatedBy: &actor.ID, Version: 1}
		signTransfer(s.Keys, &t)
		for _, id := range []int64{in.FromSpace, in.ToSpace} {
			if err := writeRecord(ctx, dbs[id], byID[id].UUID, t, "confirmed", nil, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	t.Status = transferOK
	return &t, nil
}

// Update changes a transfer; nil fields stay as they are.
type Update struct {
	FromAccount *int64
	FromAmount  *decimal.Decimal
	ToAccount   *int64
	ToAmount    *decimal.Decimal
	Date        *string
	Description *string
	// HasDescription applies Description even when nil, which clears it.
	HasDescription bool
}

// counterpart finds the other space of t as seen from spaceID.
func (s Service) counterpart(ctx context.Context, spaceID int64, t Transfer) (spaces.Space, spaces.Space, error) {
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return spaces.Space{}, spaces.Space{}, err
	}
	own := byID[spaceID]
	otherUUID := t.To.SpaceUUID
	if own.UUID == t.To.SpaceUUID {
		otherUUID = t.From.SpaceUUID
	}
	other, ok := byUUID[otherUUID]
	if !ok {
		return own, spaces.Space{}, ErrNotLinked
	}
	if linked, err := s.Linked(ctx, own.ID, other.ID); err != nil {
		return own, other, err
	} else if !linked {
		return own, other, ErrNotLinked
	}
	return own, other, nil
}

// change rewrites a transfer in both spaces with the next version; edit may
// change it within the transaction (nil deletes it).
func (s Service) change(ctx context.Context, actor *auth.User, spaceID int64, id string, edit func(map[int64]store.DB, spaces.Space, spaces.Space, *Transfer) error) (*Transfer, error) {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return nil, err
	}
	if cur.DeletedAt != nil {
		return nil, ErrGone
	}
	if cur.Status == transferPending {
		return nil, ErrPending
	}
	if !verifyTransfer(s.Keys, *cur) {
		return nil, ErrUnverified
	}
	own, other, err := s.counterpart(ctx, spaceID, *cur)
	if err != nil {
		return nil, err
	}
	if err := s.requireEditor(ctx, actor, own.ID, other.ID); err != nil {
		return nil, err
	}
	var out Transfer
	err = s.st().InSpaces(ctx, []int64{own.ID, other.ID}, func(dbs map[int64]store.DB) error {
		next, err := loadTransfer(ctx, dbs[own.ID], id) // under the locks
		if err != nil {
			return err
		}
		if theirs, err := loadTransfer(ctx, dbs[other.ID], id); err == nil && theirs.Version > next.Version {
			next.Version = theirs.Version
		}
		next.Version++
		next.Status, next.Review, next.Remote = "", nil, nil
		if err := edit(dbs, own, other, next); err != nil {
			return err
		}
		signTransfer(s.Keys, next)
		for _, sp := range []spaces.Space{own, other} {
			if err := writeRecord(ctx, dbs[sp.ID], sp.UUID, *next, "confirmed", nil, nil); err != nil {
				return err
			}
		}
		out = *next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes the amounts, accounts, date or description of a transfer.
func (s Service) Update(ctx context.Context, actor *auth.User, spaceID int64, id string, in Update) (*Transfer, error) {
	return s.change(ctx, actor, spaceID, id, func(dbs map[int64]store.DB, own, other spaces.Space, t *Transfer) error {
		byUUID := map[string]int64{own.UUID: own.ID, other.UUID: other.ID}
		apply := func(side *Side, account *int64, amount *decimal.Decimal) error {
			acc, amt := side.AccountID, decimal.New(side.Amount, -int32(side.Decimals))
			if account != nil {
				acc = *account
			}
			if amount != nil {
				amt = *amount
			}
			next, err := sideFor(ctx, dbs[byUUID[side.SpaceUUID]], side.SpaceUUID, acc, amt)
			if err != nil {
				return err
			}
			*side = next
			return nil
		}
		if err := apply(&t.From, in.FromAccount, in.FromAmount); err != nil {
			return err
		}
		if err := apply(&t.To, in.ToAccount, in.ToAmount); err != nil {
			return err
		}
		if in.Date != nil {
			t.Date = *in.Date
		}
		if in.HasDescription {
			t.Description = in.Description
		}
		return nil
	})
}

// Delete removes a transfer from both spaces (a tombstone stays).
func (s Service) Delete(ctx context.Context, actor *auth.User, spaceID int64, id string) error {
	_, err := s.change(ctx, actor, spaceID, id, func(_ map[int64]store.DB, _, _ spaces.Space, t *Transfer) error {
		at := now()
		t.DeletedAt = &at
		return nil
	})
	return err
}

// --- listing ---------------------------------------------------------------

// View is a transfer as a member of one space sees it.
type View struct {
	Transfer
	// Frozen: the other space is not linked here (unlinked, deleted, moved),
	// so the transfer can be neither changed nor merged.
	Frozen bool
	// Verified: signed by this server or a key it trusts.
	Verified   bool
	OtherSpace *spaces.Space // nil when the other space is not on this server
	Outgoing   bool
}

// List lists the live transfers of a space, newest first.
func (s Service) List(ctx context.Context, spaceID int64) ([]View, error) {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	rows, err := repo.Space{DB: d}.Records(ctx)
	if err != nil {
		return nil, err
	}
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	own := byID[spaceID]
	out := []View{}
	for _, r := range rows {
		t := transferFromRecord(r)
		if t.DeletedAt != nil && t.Status != transferPending {
			continue
		}
		v := View{Transfer: t, Verified: verifyTransfer(s.Keys, t), Outgoing: t.From.SpaceUUID == own.UUID}
		otherUUID := t.To.SpaceUUID
		if !v.Outgoing {
			otherUUID = t.From.SpaceUUID
		}
		if other, ok := byUUID[otherUUID]; ok {
			o := other
			v.OtherSpace = &o
			linked, _ := s.Linked(ctx, own.ID, other.ID)
			v.Frozen = !linked
		} else {
			v.Frozen = true
		}
		out = append(out, v)
	}
	return out, nil
}

// --- merging ---------------------------------------------------------------

// winner picks between two different versions of one record: the higher
// version, and between equal versions the one with the greater content hash,
// so both sides choose alike (P23).
func winner(a, b Transfer) Transfer {
	if a.Version != b.Version {
		if a.Version > b.Version {
			return a
		}
		return b
	}
	pa, _ := a.payload(1)
	pb, _ := b.payload(1)
	ha, hb := sha256.Sum256(pa), sha256.Sum256(pb)
	if bytes.Compare(ha[:], hb[:]) >= 0 {
		return a
	}
	return b
}

// sync merges the transfers of two spaces. With review set to one of them
// (a space just restored) differences become pending records in that space
// only; otherwise the newer verified version is copied to the side behind.
func (s Service) sync(ctx context.Context, a, b, review int64) error {
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	ua, ub := byID[a].UUID, byID[b].UUID
	return s.st().InSpaces(ctx, []int64{a, b}, func(dbs map[int64]store.DB) error {
		ra, err := loadTransfers(ctx, dbs[a])
		if err != nil {
			return err
		}
		rb, err := loadTransfers(ctx, dbs[b])
		if err != nil {
			return err
		}
		ids := map[string]bool{}
		for id := range ra {
			ids[id] = true
		}
		for id := range rb {
			ids[id] = true
		}
		keys := make([]string, 0, len(ids))
		for id := range ids {
			keys = append(keys, id)
		}
		slices.Sort(keys)
		for _, id := range keys {
			x, inA := ra[id]
			y, inB := rb[id]
			rec := x
			if !inA {
				rec = y
			}
			if !rec.involves(ua, ub) {
				continue
			}
			if review != 0 {
				own, ownUUID, mine, theirs, haveMine, haveTheirs := dbs[a], ua, x, y, inA, inB
				if review == b {
					own, ownUUID, mine, theirs, haveMine, haveTheirs = dbs[b], ub, y, x, inB, inA
				}
				if err := s.reviewOne(ctx, own, ownUUID, mine, theirs, haveMine, haveTheirs); err != nil {
					return err
				}
				continue
			}
			if (inA && x.Status == transferPending) || (inB && y.Status == transferPending) {
				continue // waits for a decision; never moved automatically
			}
			switch {
			case inA && !inB:
				if verifyTransfer(s.Keys, x) {
					if err := writeRecord(ctx, dbs[b], ub, x, "confirmed", nil, nil); err != nil {
						return err
					}
				}
			case inB && !inA:
				if verifyTransfer(s.Keys, y) {
					if err := writeRecord(ctx, dbs[a], ua, y, "confirmed", nil, nil); err != nil {
						return err
					}
				}
			case !x.same(y):
				w := winner(x, y)
				if !verifyTransfer(s.Keys, w) {
					continue
				}
				target, targetUUID := dbs[b], ub
				if w.same(y) {
					target, targetUUID = dbs[a], ua
				}
				if err := writeRecord(ctx, target, targetUUID, w, "confirmed", nil, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// reviewOne records, in a restored space, how its copy of a transfer differs
// from the other space's. Nothing is written to the other space.
func (s Service) reviewOne(ctx context.Context, own store.DB, ownUUID string, mine, theirs Transfer, haveMine, haveTheirs bool) error {
	mark := func(t Transfer, kind string, remote *Transfer, local string) error {
		k := kind
		return writeRecord(ctx, own, ownUUID, t, local, &k, remote)
	}
	switch {
	case !haveMine && haveTheirs:
		if theirs.DeletedAt != nil || !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		// Created after the backup: here as pending, outside the balance.
		return mark(theirs, reviewCreatedRemote, nil, "pending")
	case haveMine && !haveTheirs:
		if mine.DeletedAt != nil {
			return nil
		}
		return mark(mine, reviewMissingRemote, nil, "confirmed")
	case mine.same(theirs) && mine.Version == theirs.Version:
		return nil
	case theirs.DeletedAt != nil && mine.DeletedAt == nil:
		if !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		r := theirs
		return mark(mine, reviewDeletedRemote, &r, "confirmed")
	default:
		if !verifyTransfer(s.Keys, theirs) {
			return nil
		}
		r := theirs
		if mine.DeletedAt != nil {
			// Deleted in the backup, alive there: the record is kept as the
			// tombstone; the other version waits as remote.
			return writeRecord(ctx, own, ownUUID, mine, "confirmed", ptr(reviewChangedRemote), &r)
		}
		return mark(mine, reviewChangedRemote, &r, "confirmed")
	}
}

func ptr[T any](v T) *T { return &v }

// SyncAll merges every link, as at startup. A space marked as restored is
// merged in review mode, and its mark cleared once all its links are done.
func (s Service) SyncAll(ctx context.Context) error {
	links, err := s.server().AllLinks(ctx)
	if err != nil {
		return err
	}
	marked := map[int64]bool{}
	failed := map[int64]bool{}
	for _, l := range links {
		for _, id := range []int64{l.A, l.B} {
			if _, seen := marked[id]; !seen {
				marked[id] = s.reviewPending(ctx, id)
			}
		}
	}
	var errs []error
	for _, l := range links {
		review := int64(0)
		switch {
		case marked[l.A]:
			review = l.A
		case marked[l.B]:
			review = l.B
		}
		if err := s.sync(ctx, l.A, l.B, review); err != nil {
			errs = append(errs, fmt.Errorf("link %d-%d: %w", l.A, l.B, err))
			failed[l.A], failed[l.B] = true, true
		}
	}
	for id, m := range marked {
		if m && !failed[id] {
			if err := s.clearReview(ctx, id); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (s Service) reviewPending(ctx context.Context, id int64) bool {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return false
	}
	pending, err := repo.Space{DB: d}.ReviewPending(ctx)
	return err == nil && pending
}

func (s Service) clearReview(ctx context.Context, id int64) error {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return err
	}
	return repo.Space{DB: d}.ClearReview(ctx)
}

// AfterRestore merges a restored space with every space it is linked to, in
// review mode (P24); the review mark written before the restore went live is
// cleared when all succeeded, so a crash midway reviews again at startup.
// Records that do not involve this space at all are dropped first (P51).
func (s Service) AfterRestore(ctx context.Context, spaceID int64) error {
	links, err := s.Links(ctx, spaceID)
	if err != nil {
		return err
	}
	if err := s.dropForeign(ctx, spaceID); err != nil {
		return err
	}
	var errs []error
	for _, other := range links {
		if err := s.sync(ctx, spaceID, other.ID, spaceID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return s.clearReview(ctx, spaceID)
}

// dropForeign removes transfer records that do not involve the space at all:
// genuine records of other spaces slipped into its backup (P51).
func (s Service) dropForeign(ctx context.Context, spaceID int64) error {
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	own := byID[spaceID].UUID
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	return store.Tx(ctx, d, func(tx store.DB) error {
		all, err := loadTransfers(ctx, tx)
		if err != nil {
			return err
		}
		for id, t := range all {
			if _, _, ok := t.side(own); !ok {
				if err := (repo.Space{DB: tx}).Delete(ctx, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// --- deciding on a review --------------------------------------------------

// Resolve decides on a pending transfer of a restored space. Accepting takes
// the other space's state into this one (editor here). Rejecting pushes this
// space's state to the other one with a new version (editor in both); a
// record whose signature this server cannot verify is never pushed (P43).
func (s Service) Resolve(ctx context.Context, actor *auth.User, spaceID int64, id string, accept bool) error {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return err
	}
	if cur.Status != transferPending || cur.Review == nil {
		return ErrNotUnderReview
	}
	own, other, err := s.counterpart(ctx, spaceID, *cur)
	if err != nil {
		return err
	}
	if accept {
		if err := s.requireEditor(ctx, actor, own.ID); err != nil {
			return err
		}
	} else {
		if err := s.requireEditor(ctx, actor, own.ID, other.ID); err != nil {
			return err
		}
		if *cur.Review != reviewCreatedRemote && !verifyTransfer(s.Keys, *cur) {
			return ErrUnverified
		}
	}
	return s.st().InSpaces(ctx, []int64{own.ID, other.ID}, func(dbs map[int64]store.DB) error {
		mine := dbs[own.ID]
		cur, err := loadTransfer(ctx, mine, id)
		if err != nil {
			return err
		}
		theirs, theirsErr := loadTransfer(ctx, dbs[other.ID], id)
		if theirsErr != nil && !errors.Is(theirsErr, ErrGone) {
			return theirsErr
		}
		if accept {
			switch *cur.Review {
			case reviewCreatedRemote:
				cur.Review, cur.Remote = nil, nil
				return writeRecord(ctx, mine, own.UUID, *cur, "confirmed", nil, nil)
			case reviewMissingRemote:
				return repo.Space{DB: mine}.Delete(ctx, id)
			default:
				if theirs == nil {
					return ErrGone
				}
				return writeRecord(ctx, mine, own.UUID, *theirs, "confirmed", nil, nil)
			}
		}
		next := *cur
		next.Review, next.Remote = nil, nil
		version := cur.Version
		if theirs != nil && theirs.Version > version {
			version = theirs.Version
		}
		next.Version = version + 1
		if *cur.Review == reviewCreatedRemote {
			at := now()
			next.DeletedAt = &at
		}
		signTransfer(s.Keys, &next)
		for _, sp := range []spaces.Space{own, other} {
			if err := writeRecord(ctx, dbs[sp.ID], sp.UUID, next, "confirmed", nil, nil); err != nil {
				return err
			}
		}
		return nil
	})
}

// Trust re-signs a transfer this server cannot verify (made on a server whose
// key it does not trust), after its members agree (P46): editor in every
// space of it that is on this server.
func (s Service) Trust(ctx context.Context, actor *auth.User, spaceID int64, id string) error {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return err
	}
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return err
	}
	spaces := []int64{spaceID}
	for _, u := range []string{cur.From.SpaceUUID, cur.To.SpaceUUID} {
		if sp, ok := byUUID[u]; ok && sp.ID != spaceID {
			spaces = append(spaces, sp.ID)
		}
	}
	if err := s.requireEditor(ctx, actor, spaces...); err != nil {
		return err
	}
	return s.st().InSpaces(ctx, spaces, func(dbs map[int64]store.DB) error {
		next, err := loadTransfer(ctx, dbs[spaceID], id)
		if err != nil {
			return err
		}
		signTransfer(s.Keys, next)
		for _, sid := range spaces {
			if err := writeRecord(ctx, dbs[sid], byID[sid].UUID, *next, "confirmed", next.Review, next.Remote); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- keys ------------------------------------------------------------------

// TrustedKey is a public key this server accepts signatures of.
type TrustedKey struct {
	KID       string
	PublicKey string
	Name      string
	CreatedAt *time.Time
}

// TrustedKeysList lists the trusted keys.
func (s Service) TrustedKeysList(ctx context.Context) ([]TrustedKey, error) {
	rows, err := s.server().TrustedKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TrustedKey, len(rows))
	for i, r := range rows {
		out[i] = TrustedKey{KID: r.KID, PublicKey: r.PublicKey, Name: r.Name, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// Trust adds a public key (base64) to the trusted keys.
func (s Service) TrustKey(ctx context.Context, actor *auth.User, name, publicKey string) (string, error) {
	pub, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return "", errors.New("the public key must be a base64 Ed25519 key")
	}
	kid := signing.KID(pub)
	if err := s.server().InsertTrustedKey(ctx, kid, publicKey, name, actor.ID, time.Now().UTC()); err != nil {
		return "", err
	}
	return kid, s.Spaces.Audit(ctx, actor, "trust_key", nil, nil, kid+" "+name)
}

// UntrustKey removes a trusted key.
func (s Service) UntrustKey(ctx context.Context, actor *auth.User, kid string) error {
	ok, err := s.server().DeleteTrustedKey(ctx, kid)
	if err != nil {
		return err
	}
	if !ok {
		return ErrGone
	}
	return s.Spaces.Audit(ctx, actor, "untrust_key", nil, nil, kid)
}

// RotateKey replaces the server key; the old public key becomes trusted, so
// what it signed still verifies (P47).
func (s Service) RotateKey(ctx context.Context, actor *auth.User, dataDir string) (string, error) {
	old := s.Keys.Holder.Key()
	if old != nil {
		pub := base64.StdEncoding.EncodeToString(old.Public())
		if err := s.server().InsertTrustedKey(ctx, old.KID(), pub, "previous server key", actor.ID, time.Now().UTC()); err != nil {
			return "", err
		}
	}
	key, err := signing.Generate(dataDir)
	if err != nil {
		return "", err
	}
	s.Keys.Holder.Set(key)
	if err := (settings.Store{DB: s.st().Server()}).Set(ctx, "signing_kid", key.KID()); err != nil {
		return "", err
	}
	return key.KID(), s.Spaces.Audit(ctx, actor, "rotate_key", nil, nil, key.KID())
}
