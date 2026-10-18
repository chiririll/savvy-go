package transfers

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/transfers"
	"savvy-go/internal/money"
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

// --- rows ------------------------------------------------------------------

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
	return Side{
		SpaceUUID: spaceUUID,
		AccountID: account,
		Amount:    m.Minor(),
		Currency:  acc.Code,
		Decimals:  acc.Decimals,
	}, nil
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
		t = Transfer{
			UUID:        id7.String(),
			From:        from,
			To:          to,
			Date:        in.Date,
			Description: in.Description,
			CreatedBy:   &actor.ID,
			Version:     1,
		}
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
func (s Service) change(
	ctx context.Context,
	actor *auth.User,
	spaceID int64,
	id string,
	edit func(map[int64]store.DB, spaces.Space, spaces.Space, *Transfer) error,
) (*Transfer, error) {
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

func ptr[T any](v T) *T { return &v }

// --- deciding on a review --------------------------------------------------

// --- keys ------------------------------------------------------------------
