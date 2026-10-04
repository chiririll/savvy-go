package domain

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
	"savvy-go/internal/signing"
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

// TransferSide is one space's side of a transfer. AccountID belongs to that
// space; the other space only stores it as an opaque number.
type TransferSide struct {
	SpaceUUID string `json:"space"`
	AccountID int64  `json:"account"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Decimals  int64  `json:"decimals"`
}

// Transfer is a transfer record.
type Transfer struct {
	UUID        string       `json:"uuid"`
	From        TransferSide `json:"from"`
	To          TransferSide `json:"to"`
	Date        string       `json:"date"`
	Description *string      `json:"description"`
	CreatedBy   *int64       `json:"created_by"`
	Version     int64        `json:"version"`
	DeletedAt   *string      `json:"deleted_at"`
	SigV        int64        `json:"sig_v"`
	SignerKID   string       `json:"signer_kid"`
	Signature   string       `json:"signature"`
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
	transferReviewKey = "transfer_review_pending"
)

var (
	ErrTransferRow     = errors.New("this transaction is one side of a transfer between spaces; change the transfer instead")
	ErrNotLinked       = errors.New("these spaces are not linked")
	ErrSameSpace       = errors.New("a transfer between spaces needs two different spaces")
	ErrTransferRights  = errors.New("you need to be an editor in both spaces")
	ErrAccountMissing  = errors.New("the account does not exist in that space")
	ErrUnverified      = errors.New("this transfer's signature cannot be verified on this server")
	ErrNotUnderReview  = errors.New("this transfer is not waiting for a decision")
	ErrTransferPending = errors.New("decide on this transfer's pending change first")
	ErrTransferGone    = errors.New("transfer not found")
)

// --- signing ---------------------------------------------------------------

// transferPayloadV1 is what sig_v 1 signs. A later format adds fields in a
// new struct and a new sig_v; records keep verifying by the format they were
// signed with (P49).
type transferPayloadV1 struct {
	UUID        string       `json:"uuid"`
	From        TransferSide `json:"from"`
	To          TransferSide `json:"to"`
	Date        string       `json:"date"`
	Description *string      `json:"description"`
	CreatedBy   *int64       `json:"created_by"`
	Version     int64        `json:"version"`
	DeletedAt   *string      `json:"deleted_at"`
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

func (k KeyRing) sign(t *Transfer) {
	key := k.Holder.Key()
	t.SigV = currentSigV
	p, _ := t.payload(currentSigV)
	t.SignerKID = key.KID()
	t.Signature = base64.StdEncoding.EncodeToString(key.Sign(p))
}

// verify reports whether t is signed by this server or a key it trusts.
func (k KeyRing) verify(t Transfer) bool {
	p, ok := t.payload(t.SigV)
	if !ok {
		return false
	}
	return k.check(t.SignerKID, p, []byte(t.Signature)) != Unsigned
}

// TrustedKeys looks up a trusted public key in the server database.
func TrustedKeys(server store.DB) func(kid string) ed25519.PublicKey {
	return func(kid string) ed25519.PublicKey {
		raw, err := db.Q(server).GetTrustedKey(context.Background(), kid)
		if err != nil {
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

func transferFromRow(r sqlc.SpaceTransfer) Transfer {
	t := Transfer{
		UUID: r.Uuid,
		From: TransferSide{r.FromSpaceUuid, r.FromAccountID, r.FromAmount, r.FromCurrency, r.FromDecimals},
		To:   TransferSide{r.ToSpaceUuid, r.ToAccountID, r.ToAmount, r.ToCurrency, r.ToDecimals},
		Date: r.Date, Version: r.Version, SigV: r.SigV, SignerKID: r.SignerKid.String, Signature: r.Signature.String,
		Status: r.Status,
	}
	if r.Description.Valid {
		v := r.Description.String
		t.Description = &v
	}
	if r.CreatedBy.Valid {
		v := r.CreatedBy.Int64
		t.CreatedBy = &v
	}
	if r.DeletedAt.Valid {
		v := r.DeletedAt.String
		t.DeletedAt = &v
	}
	if r.Review.Valid {
		v := r.Review.String
		t.Review = &v
	}
	if r.Remote.Valid {
		var remote Transfer
		if json.Unmarshal([]byte(r.Remote.String), &remote) == nil {
			t.Remote = &remote
		}
	}
	return t
}

func loadTransfers(ctx context.Context, d store.DB) (map[string]Transfer, error) {
	rows, err := db.Q(d).ListSpaceTransfers(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Transfer, len(rows))
	for _, r := range rows {
		out[r.Uuid] = transferFromRow(sqlc.SpaceTransfer(r))
	}
	return out, nil
}

func loadTransfer(ctx context.Context, d store.DB, id string) (*Transfer, error) {
	r, err := db.Q(d).GetSpaceTransfer(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTransferGone
	}
	if err != nil {
		return nil, err
	}
	t := transferFromRow(sqlc.SpaceTransfer(r))
	return &t, nil
}

// side is own's side of t, and whether own is the sender.
func (t Transfer) side(own string) (TransferSide, bool, bool) {
	switch own {
	case t.From.SpaceUUID:
		return t.From, true, true
	case t.To.SpaceUUID:
		return t.To, false, true
	}
	return TransferSide{}, false, false
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
	q := db.Q(d)
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
		if _, err := q.GetAccountCurrency(ctx, side.AccountID); err == nil {
			hasAccount = true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		} else if review == nil {
			status = transferNeedsAttention
		}
	}
	var remoteJSON sql.NullString
	if remote != nil {
		b, _ := json.Marshal(remote)
		remoteJSON = db.NS(string(b))
	}
	if err := q.UpsertSpaceTransfer(ctx, sqlc.UpsertSpaceTransferParams{
		Uuid: t.UUID, FromSpaceUuid: t.From.SpaceUUID, FromAccountID: t.From.AccountID, FromAmount: t.From.Amount,
		FromCurrency: t.From.Currency, FromDecimals: t.From.Decimals,
		ToSpaceUuid: t.To.SpaceUUID, ToAccountID: t.To.AccountID, ToAmount: t.To.Amount,
		ToCurrency: t.To.Currency, ToDecimals: t.To.Decimals,
		Date: t.Date, Description: db.NullString(t.Description), CreatedBy: db.NullInt64(t.CreatedBy),
		Version: t.Version, DeletedAt: db.NullString(t.DeletedAt), SigV: t.SigV,
		SignerKid: db.NullStringVal(t.SignerKID), Signature: db.NullStringVal(t.Signature),
		Status: status, Review: db.NullString(review), Remote: remoteJSON, UpdatedAt: db.NS(now()),
	}); err != nil {
		return err
	}
	if t.DeletedAt != nil || !hasAccount {
		return q.DeleteTransferTransaction(ctx, db.NS(t.UUID))
	}
	typ := "transfer_in"
	if sender {
		typ = "transfer_out"
	}
	if _, err := q.GetTransferTransaction(ctx, db.NS(t.UUID)); errors.Is(err, sql.ErrNoRows) {
		return q.InsertTransferTransaction(ctx, sqlc.InsertTransferTransactionParams{
			Type: typ, AccountID: side.AccountID, Amount: side.Amount, Description: db.NullString(t.Description),
			Date: db.NS(t.Date), Status: localStatus, SpaceTransferUuid: db.NS(t.UUID), CreatedBy: db.NullInt64(t.CreatedBy),
			CreatedAt: db.NS(now()), UpdatedAt: db.NS(now()),
		})
	} else if err != nil {
		return err
	}
	return q.UpdateTransferTransaction(ctx, sqlc.UpdateTransferTransactionParams{
		Type: typ, AccountID: side.AccountID, Amount: side.Amount, Description: db.NullString(t.Description),
		Date: db.NS(t.Date), Status: localStatus, UpdatedAt: db.NS(now()), SpaceTransferUuid: db.NS(t.UUID),
	})
}

// --- service ---------------------------------------------------------------

// Transfers links spaces and moves money between them.
type Transfers struct {
	Spaces Spaces
	Keys   KeyRing
}

func (s Transfers) st() store.Store       { return s.Spaces.Store }
func (s Transfers) server() *sqlc.Queries { return s.Spaces.server() }
func linkKey(a, b int64) (int64, int64)   { return min(a, b), max(a, b) }

// spaceUUIDs maps every registered space's id to its uuid and back.
func (s Transfers) spaceUUIDs(ctx context.Context) (map[int64]Space, map[string]Space, error) {
	rows, err := s.server().ListSpaceUUIDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	byID, byUUID := map[int64]Space{}, map[string]Space{}
	for _, r := range rows {
		sp := Space{ID: r.ID, UUID: r.Uuid, Name: r.Name}
		byID[r.ID], byUUID[r.Uuid] = sp, sp
	}
	return byID, byUUID, nil
}

// Linked reports whether two spaces are linked.
func (s Transfers) Linked(ctx context.Context, a, b int64) (bool, error) {
	x, y := linkKey(a, b)
	n, err := s.server().CountSpaceLink(ctx, sqlc.CountSpaceLinkParams{SpaceAID: x, SpaceBID: y})
	return n > 0, err
}

// Links lists the spaces linked to a space.
func (s Transfers) Links(ctx context.Context, spaceID int64) ([]Space, error) {
	rows, err := s.server().ListSpaceLinks(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	byID, _, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := []Space{}
	for _, r := range rows {
		other := r.SpaceAID
		if other == spaceID {
			other = r.SpaceBID
		}
		out = append(out, byID[other])
	}
	return out, nil
}

// ErrLinkRights is returned when the linker does not administer both spaces.
var ErrLinkRights = errors.New("you need to be an admin of both spaces to link them")

// Link links two spaces; only who administers both may (P52). Transfers the
// two already share (spaces moved between servers together) are merged.
func (s Transfers) Link(ctx context.Context, actor *auth.User, a, b int64) error {
	if a == b {
		return ErrSameSpace
	}
	for _, id := range []int64{a, b} {
		if role, err := s.Spaces.Role(ctx, id, actor.ID); err != nil {
			return err
		} else if role != SpaceAdmin {
			return ErrLinkRights
		}
	}
	x, y := linkKey(a, b)
	if err := s.server().InsertSpaceLink(ctx, sqlc.InsertSpaceLinkParams{SpaceAID: x, SpaceBID: y, CreatedBy: db.NI(actor.ID), CreatedAt: db.NS(now())}); err != nil {
		return err
	}
	return s.sync(ctx, a, b, 0)
}

// Unlink removes a link; an admin of either space may. Transfers stay in
// both spaces, frozen.
func (s Transfers) Unlink(ctx context.Context, a, b int64) error {
	x, y := linkKey(a, b)
	res, err := s.server().DeleteSpaceLink(ctx, sqlc.DeleteSpaceLinkParams{SpaceAID: x, SpaceBID: y})
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotLinked
	}
	return nil
}

func (s Transfers) requireEditor(ctx context.Context, actor *auth.User, ids ...int64) error {
	for _, id := range ids {
		role, err := s.Spaces.Role(ctx, id, actor.ID)
		if err != nil {
			return err
		}
		if !CanWriteSpace(role) {
			return ErrTransferRights
		}
	}
	return nil
}

// TransferInput is a transfer as entered: amounts in each account's currency.
type TransferInput struct {
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
func sideFor(ctx context.Context, d store.DB, spaceUUID string, account int64, amount decimal.Decimal) (TransferSide, error) {
	acc, err := db.Q(d).GetAccountCurrency(ctx, account)
	if errors.Is(err, sql.ErrNoRows) {
		return TransferSide{}, ErrAccountMissing
	}
	if err != nil {
		return TransferSide{}, err
	}
	m, err := money.FromInput(amount, money.Unit{Decimals: int(acc.Decimals)})
	if err != nil {
		return TransferSide{}, err
	}
	if m.Minor() <= 0 {
		return TransferSide{}, errors.New("the amounts must be positive")
	}
	return TransferSide{SpaceUUID: spaceUUID, AccountID: account, Amount: m.Minor(), Currency: acc.Code, Decimals: acc.Decimals}, nil
}

// Create records a transfer in both spaces at once.
func (s Transfers) Create(ctx context.Context, actor *auth.User, in TransferInput) (*Transfer, error) {
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
		s.Keys.sign(&t)
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

// TransferUpdate changes a transfer; nil fields stay as they are.
type TransferUpdate struct {
	FromAccount *int64
	FromAmount  *decimal.Decimal
	ToAccount   *int64
	ToAmount    *decimal.Decimal
	Date        *string
	Description *string
}

// counterpart finds the other space of t as seen from spaceID.
func (s Transfers) counterpart(ctx context.Context, spaceID int64, t Transfer) (Space, Space, error) {
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return Space{}, Space{}, err
	}
	own := byID[spaceID]
	otherUUID := t.To.SpaceUUID
	if own.UUID == t.To.SpaceUUID {
		otherUUID = t.From.SpaceUUID
	}
	other, ok := byUUID[otherUUID]
	if !ok {
		return own, Space{}, ErrNotLinked
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
func (s Transfers) change(ctx context.Context, actor *auth.User, spaceID int64, id string, edit func(map[int64]store.DB, Space, Space, *Transfer) error) (*Transfer, error) {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	cur, err := loadTransfer(ctx, d, id)
	if err != nil {
		return nil, err
	}
	if cur.DeletedAt != nil {
		return nil, ErrTransferGone
	}
	if cur.Status == transferPending {
		return nil, ErrTransferPending
	}
	if !s.Keys.verify(*cur) {
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
		s.Keys.sign(next)
		for _, sp := range []Space{own, other} {
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
func (s Transfers) Update(ctx context.Context, actor *auth.User, spaceID int64, id string, in TransferUpdate) (*Transfer, error) {
	return s.change(ctx, actor, spaceID, id, func(dbs map[int64]store.DB, own, other Space, t *Transfer) error {
		byUUID := map[string]int64{own.UUID: own.ID, other.UUID: other.ID}
		apply := func(side *TransferSide, account *int64, amount *decimal.Decimal) error {
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
		if in.Description != nil {
			t.Description = in.Description
		}
		return nil
	})
}

// Delete removes a transfer from both spaces (a tombstone stays).
func (s Transfers) Delete(ctx context.Context, actor *auth.User, spaceID int64, id string) error {
	_, err := s.change(ctx, actor, spaceID, id, func(_ map[int64]store.DB, _, _ Space, t *Transfer) error {
		at := now()
		t.DeletedAt = &at
		return nil
	})
	return err
}

// --- listing ---------------------------------------------------------------

// TransferView is a transfer as a member of one space sees it.
type TransferView struct {
	Transfer
	// Frozen: the other space is not linked here (unlinked, deleted, moved),
	// so the transfer can be neither changed nor merged.
	Frozen bool
	// Verified: signed by this server or a key it trusts.
	Verified   bool
	OtherSpace *Space // nil when the other space is not on this server
	Outgoing   bool
}

// List lists the live transfers of a space, newest first.
func (s Transfers) List(ctx context.Context, spaceID int64) ([]TransferView, error) {
	d, err := s.st().Space(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Q(d).ListSpaceTransfers(ctx)
	if err != nil {
		return nil, err
	}
	byID, byUUID, err := s.spaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	own := byID[spaceID]
	out := []TransferView{}
	for _, r := range rows {
		t := transferFromRow(sqlc.SpaceTransfer(r))
		if t.DeletedAt != nil && t.Status != transferPending {
			continue
		}
		v := TransferView{Transfer: t, Verified: s.Keys.verify(t), Outgoing: t.From.SpaceUUID == own.UUID}
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
func (s Transfers) sync(ctx context.Context, a, b, review int64) error {
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
				if s.Keys.verify(x) {
					if err := writeRecord(ctx, dbs[b], ub, x, "confirmed", nil, nil); err != nil {
						return err
					}
				}
			case inB && !inA:
				if s.Keys.verify(y) {
					if err := writeRecord(ctx, dbs[a], ua, y, "confirmed", nil, nil); err != nil {
						return err
					}
				}
			case !x.same(y):
				w := winner(x, y)
				if !s.Keys.verify(w) {
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
func (s Transfers) reviewOne(ctx context.Context, own store.DB, ownUUID string, mine, theirs Transfer, haveMine, haveTheirs bool) error {
	mark := func(t Transfer, kind string, remote *Transfer, local string) error {
		k := kind
		return writeRecord(ctx, own, ownUUID, t, local, &k, remote)
	}
	switch {
	case !haveMine && haveTheirs:
		if theirs.DeletedAt != nil || !s.Keys.verify(theirs) {
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
		if !s.Keys.verify(theirs) {
			return nil
		}
		r := theirs
		return mark(mine, reviewDeletedRemote, &r, "confirmed")
	default:
		if !s.Keys.verify(theirs) {
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
func (s Transfers) SyncAll(ctx context.Context) error {
	links, err := s.server().ListAllSpaceLinks(ctx)
	if err != nil {
		return err
	}
	marked := map[int64]bool{}
	failed := map[int64]bool{}
	for _, l := range links {
		for _, id := range []int64{l.SpaceAID, l.SpaceBID} {
			if _, seen := marked[id]; !seen {
				marked[id] = s.reviewPending(ctx, id)
			}
		}
	}
	var errs []error
	for _, l := range links {
		review := int64(0)
		switch {
		case marked[l.SpaceAID]:
			review = l.SpaceAID
		case marked[l.SpaceBID]:
			review = l.SpaceBID
		}
		if err := s.sync(ctx, l.SpaceAID, l.SpaceBID, review); err != nil {
			errs = append(errs, fmt.Errorf("link %d-%d: %w", l.SpaceAID, l.SpaceBID, err))
			failed[l.SpaceAID], failed[l.SpaceBID] = true, true
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

func (s Transfers) reviewPending(ctx context.Context, id int64) bool {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return false
	}
	v, err := db.Q(d).GetSpaceSetting(ctx, transferReviewKey)
	return err == nil && v.Valid && v.String != ""
}

func (s Transfers) clearReview(ctx context.Context, id int64) error {
	d, err := s.st().Space(ctx, id)
	if err != nil {
		return err
	}
	_, err = d.ExecContext(ctx, `DELETE FROM space_settings WHERE key = ?`, transferReviewKey)
	return err
}

// AfterRestore merges a restored space with every space it is linked to, in
// review mode (P24); the review mark written before the restore went live is
// cleared when all succeeded, so a crash midway reviews again at startup.
// Records that do not involve this space at all are dropped first (P51).
func (s Transfers) AfterRestore(ctx context.Context, spaceID int64) error {
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
func (s Transfers) dropForeign(ctx context.Context, spaceID int64) error {
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
				if err := db.Q(tx).DeleteTransferTransaction(ctx, db.NS(id)); err != nil {
					return err
				}
				if err := db.Q(tx).DeleteSpaceTransfer(ctx, id); err != nil {
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
func (s Transfers) Resolve(ctx context.Context, actor *auth.User, spaceID int64, id string, accept bool) error {
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
		if *cur.Review != reviewCreatedRemote && !s.Keys.verify(*cur) {
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
		if theirsErr != nil && !errors.Is(theirsErr, ErrTransferGone) {
			return theirsErr
		}
		if accept {
			switch *cur.Review {
			case reviewCreatedRemote:
				cur.Review, cur.Remote = nil, nil
				return writeRecord(ctx, mine, own.UUID, *cur, "confirmed", nil, nil)
			case reviewMissingRemote:
				if err := db.Q(mine).DeleteTransferTransaction(ctx, db.NS(id)); err != nil {
					return err
				}
				return db.Q(mine).DeleteSpaceTransfer(ctx, id)
			default:
				if theirs == nil {
					return ErrTransferGone
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
		s.Keys.sign(&next)
		for _, sp := range []Space{own, other} {
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
func (s Transfers) Trust(ctx context.Context, actor *auth.User, spaceID int64, id string) error {
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
		s.Keys.sign(next)
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
func (s Transfers) TrustedKeysList(ctx context.Context) ([]TrustedKey, error) {
	rows, err := s.server().ListTrustedKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TrustedKey, len(rows))
	for i, r := range rows {
		out[i] = TrustedKey{KID: r.Kid, PublicKey: r.PublicKey, Name: r.Name}
		if t, ok := parseNullTime(r.CreatedAt); ok {
			out[i].CreatedAt = &t
		}
	}
	return out, nil
}

// Trust adds a public key (base64) to the trusted keys.
func (s Transfers) TrustKey(ctx context.Context, actor *auth.User, name, publicKey string) (string, error) {
	pub, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return "", errors.New("the public key must be a base64 Ed25519 key")
	}
	kid := signing.KID(pub)
	if err := s.server().InsertTrustedKey(ctx, sqlc.InsertTrustedKeyParams{
		Kid: kid, PublicKey: publicKey, Name: name, AddedBy: db.NI(actor.ID), CreatedAt: db.NS(now()),
	}); err != nil {
		return "", err
	}
	return kid, s.Spaces.Audit(ctx, actor, "trust_key", nil, nil, kid+" "+name)
}

// UntrustKey removes a trusted key.
func (s Transfers) UntrustKey(ctx context.Context, actor *auth.User, kid string) error {
	res, err := s.server().DeleteTrustedKey(ctx, kid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTransferGone
	}
	return s.Spaces.Audit(ctx, actor, "untrust_key", nil, nil, kid)
}

// RotateKey replaces the server key; the old public key becomes trusted, so
// what it signed still verifies (P47).
func (s Transfers) RotateKey(ctx context.Context, actor *auth.User, dataDir string) (string, error) {
	old := s.Keys.Holder.Key()
	if old != nil {
		pub := base64.StdEncoding.EncodeToString(old.Public())
		if err := s.server().InsertTrustedKey(ctx, sqlc.InsertTrustedKeyParams{
			Kid: old.KID(), PublicKey: pub, Name: "previous server key", AddedBy: db.NI(actor.ID), CreatedAt: db.NS(now()),
		}); err != nil {
			return "", err
		}
	}
	key, err := signing.Generate(dataDir)
	if err != nil {
		return "", err
	}
	s.Keys.Holder.Set(key)
	if err := s.Spaces.settings().Set(ctx, "signing_kid", key.KID()); err != nil {
		return "", err
	}
	return key.KID(), s.Spaces.Audit(ctx, actor, "rotate_key", nil, nil, key.KID())
}
