// Package spacesrepo is the registry of spaces in the server database: the
// spaces, their members, invitations and the admin audit log. Package spaces
// owns the rules (roles, limits, who may do what).
//
// A Repo works on any handle, so a method used inside store.Tx gets the
// transaction's handle: spacesrepo.Repo{DB: tx}.
package spaces

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

type Repo struct{ DB store.DB }

func (r Repo) q() *sqlc.Queries { return db.Q(r.DB) }

func optTime(raw db.NString) *time.Time {
	if t, ok := db.ParseTime(raw); ok {
		return &t
	}
	return nil
}

func timePtr(raw string) *time.Time {
	if t, ok := db.ParseTimeString(raw); ok {
		return &t
	}
	return nil
}

// Spaces

type Space struct {
	ID         int64
	UUID, Name string
}

// UserSpace is a space with the role of the user it was listed for.
type UserSpace struct {
	Space
	Role string
}

// Insert registers a space and returns its id.
func (r Repo) Insert(ctx context.Context, uuid, name string, createdBy *int64, now time.Time) (int64, error) {
	return r.q().InsertSpace(ctx, sqlc.InsertSpaceParams{
		Uuid: uuid, Name: name, CreatedBy: db.NullInt64(createdBy), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

func (r Repo) Delete(ctx context.Context, id int64) error { return r.q().DeleteSpaceRow(ctx, id) }

// HasUUID reports whether a space with that uuid is registered.
func (r Repo) HasUUID(ctx context.Context, uuid string) (bool, error) {
	_, err := r.q().SpaceIDByUUID(ctx, uuid)
	if err == db.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// ByID returns nil for an unknown space.
func (r Repo) ByID(ctx context.Context, id int64) (*Space, error) {
	row, err := r.q().GetSpace(ctx, id)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Space{ID: row.ID, UUID: row.Uuid, Name: row.Name}, nil
}

func (r Repo) IDs(ctx context.Context) ([]int64, error) { return r.q().ListSpaceIDs(ctx) }

func (r Repo) Rename(ctx context.Context, id int64, name string, now time.Time) error {
	return r.q().RenameSpace(ctx, sqlc.RenameSpaceParams{Name: name, UpdatedAt: db.TimeNS(now), ID: id})
}

// Quota is a space's own size limit in bytes; nil when it has none.
func (r Repo) Quota(ctx context.Context, id int64) (*int64, error) {
	q, err := r.q().GetSpaceQuota(ctx, id)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if !q.Valid {
		return nil, nil
	}
	return &q.Int64, nil
}

// SetQuota sets a space's own size limit; nil clears it.
func (r Repo) SetQuota(ctx context.Context, id int64, bytes *int64, now time.Time) error {
	return r.q().SetSpaceQuota(ctx, sqlc.SetSpaceQuotaParams{QuotaBytes: db.NullInt64(bytes), UpdatedAt: db.TimeNS(now), ID: id})
}

// Overview is one row of the admin overview: counts and metadata only.
type Overview struct {
	Space
	Members, Admins int64
	CreatedAt       *time.Time
}

func (r Repo) Overview(ctx context.Context) ([]Overview, error) {
	rows, err := r.q().ListSpacesOverview(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Overview, len(rows))
	for i, row := range rows {
		out[i] = Overview{Space: Space{ID: row.ID, UUID: row.Uuid, Name: row.Name},
			Members: row.Members, Admins: row.Admins, CreatedAt: optTime(row.CreatedAt)}
	}
	return out, nil
}

// Members

type Member struct {
	UserID      int64
	Name, Email string
	Role        string
	CreatedAt   *time.Time
}

func (r Repo) UpsertMember(ctx context.Context, spaceID, userID int64, role string, now time.Time) error {
	return r.q().UpsertSpaceMember(ctx, sqlc.UpsertSpaceMemberParams{SpaceID: spaceID, UserID: userID, Role: role, CreatedAt: db.TimeNS(now)})
}

func (r Repo) DeleteMember(ctx context.Context, spaceID, userID int64) error {
	return r.q().DeleteSpaceMember(ctx, sqlc.DeleteSpaceMemberParams{SpaceID: spaceID, UserID: userID})
}

// MemberRole is the user's role in the space; "" when they are not a member.
func (r Repo) MemberRole(ctx context.Context, spaceID, userID int64) (string, error) {
	role, err := r.q().GetMemberRole(ctx, sqlc.GetMemberRoleParams{SpaceID: spaceID, UserID: userID})
	if err == db.ErrNotFound {
		return "", nil
	}
	return role, err
}

func (r Repo) Members(ctx context.Context, spaceID int64) ([]Member, error) {
	rows, err := r.q().ListMembers(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, row := range rows {
		out[i] = Member{UserID: row.UserID, Name: row.Name, Email: row.Email, Role: row.Role, CreatedAt: optTime(row.CreatedAt)}
	}
	return out, nil
}

func (r Repo) CountMembers(ctx context.Context, spaceID int64) (int64, error) {
	return r.q().CountSpaceMembers(ctx, spaceID)
}

func (r Repo) CountWithRole(ctx context.Context, spaceID int64, role string) (int64, error) {
	return r.q().CountSpaceRole(ctx, sqlc.CountSpaceRoleParams{SpaceID: spaceID, Role: role})
}

// ForUser lists the spaces a user belongs to, oldest first.
func (r Repo) ForUser(ctx context.Context, userID int64) ([]UserSpace, error) {
	rows, err := r.q().ListUserSpaces(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]UserSpace, len(rows))
	for i, row := range rows {
		out[i] = UserSpace{Space: Space{ID: row.ID, UUID: row.Uuid, Name: row.Name}, Role: row.Role}
	}
	return out, nil
}

// AdminSpaceIDs lists the spaces the user is admin of.
func (r Repo) AdminSpaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	return r.q().ListUserAdminSpaces(ctx, userID)
}

func (r Repo) CountAdminSpaces(ctx context.Context, userID int64) (int64, error) {
	return r.q().CountUserAdminSpaces(ctx, userID)
}

// Audit log

type AuditEntry struct {
	ID            int64
	Actor, Action string
	TargetUserID  *int64
	Details       string
	CreatedAt     *time.Time
}

// InsertAudit records an admin action; actor, spaceID and target are optional.
func (r Repo) InsertAudit(ctx context.Context, actor, spaceID, target *int64, action, details string, now time.Time) error {
	return r.q().InsertAudit(ctx, sqlc.InsertAuditParams{
		ActorID: db.NullInt64(actor), Action: action, SpaceID: db.NullInt64(spaceID), TargetUserID: db.NullInt64(target),
		Details: db.NullStringVal(details), CreatedAt: db.FormatTime(now),
	})
}

// Audit lists the actions on a space, newest first.
func (r Repo) Audit(ctx context.Context, spaceID int64) ([]AuditEntry, error) {
	rows, err := r.q().ListSpaceAudit(ctx, db.NI(spaceID))
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, len(rows))
	for i, row := range rows {
		e := AuditEntry{ID: row.ID, Actor: row.Actor, Action: row.Action, Details: row.Details.String, CreatedAt: timePtr(row.CreatedAt)}
		if row.TargetUserID.Valid {
			id := row.TargetUserID.Int64
			e.TargetUserID = &id
		}
		out[i] = e
	}
	return out, nil
}

// Invitations

type NewInvitation struct {
	SpaceID, InvitedBy int64
	Email              string // empty: open to anyone
	Role, TokenHash    string
	ByServerAdmin      bool
	Expires, Now       time.Time
}

// InsertInvitation stores an invitation and returns its id.
func (r Repo) InsertInvitation(ctx context.Context, n NewInvitation) (int64, error) {
	return r.q().InsertInvitation(ctx, sqlc.InsertInvitationParams{
		SpaceID: n.SpaceID, Email: db.NullStringVal(n.Email), Role: n.Role, TokenHash: n.TokenHash,
		InvitedBy: db.NI(n.InvitedBy), InvitedByServerAdmin: db.BoolInt(n.ByServerAdmin),
		ExpiresAt: db.FormatTime(n.Expires), CreatedAt: db.TimeNS(n.Now),
	})
}

// Invitation is one as a space's admins list it.
type Invitation struct {
	ID                               int64
	Email                            *string
	Role, Inviter                    string
	ExpiresAt, AcceptedAt, CreatedAt *time.Time
}

func (r Repo) Invitations(ctx context.Context, spaceID int64) ([]Invitation, error) {
	rows, err := r.q().ListInvitations(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Invitation, len(rows))
	for i, row := range rows {
		inv := Invitation{ID: row.ID, Role: row.Role, Inviter: row.Inviter,
			ExpiresAt: timePtr(row.ExpiresAt), AcceptedAt: optTime(row.AcceptedAt), CreatedAt: optTime(row.CreatedAt)}
		if row.Email.Valid {
			inv.Email = &row.Email.String
		}
		out[i] = inv
	}
	return out, nil
}

// DeleteInvitation reports whether the space had such an invitation.
func (r Repo) DeleteInvitation(ctx context.Context, spaceID, id int64) (bool, error) {
	res, err := r.q().DeleteInvitation(ctx, sqlc.DeleteInvitationParams{ID: id, SpaceID: spaceID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// InvitationByHash is an invitation found by its token's hash, with the space
// and inviter it names.
type InvitationByHash struct {
	ID, SpaceID         int64
	Email               *string
	Role                string
	ByServerAdmin, Used bool
	Expires             time.Time
	SpaceName, Inviter  string
}

// InvitationByHash returns nil for an unknown token.
func (r Repo) InvitationByHash(ctx context.Context, hash string) (*InvitationByHash, error) {
	row, err := r.q().GetInvitationByHash(ctx, hash)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	inv := &InvitationByHash{ID: row.ID, SpaceID: row.SpaceID, Role: row.Role, ByServerAdmin: row.InvitedByServerAdmin == 1,
		Used: row.AcceptedAt.Valid, SpaceName: row.SpaceName, Inviter: row.Inviter}
	if row.Email.Valid {
		inv.Email = &row.Email.String
	}
	inv.Expires, _ = db.ParseTimeString(row.ExpiresAt)
	return inv, nil
}

// AcceptInvitation marks an unused invitation used and reports whether it did.
func (r Repo) AcceptInvitation(ctx context.Context, id, userID int64, now time.Time) (bool, error) {
	res, err := r.q().AcceptInvitation(ctx, sqlc.AcceptInvitationParams{AcceptedAt: db.TimeNS(now), AcceptedBy: db.NI(userID), ID: id})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// All lists every registered space with its uuid.
func (r Repo) All(ctx context.Context) ([]Space, error) {
	rows, err := r.q().ListSpaceUUIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Space, len(rows))
	for i, row := range rows {
		out[i] = Space{ID: row.ID, UUID: row.Uuid, Name: row.Name}
	}
	return out, nil
}

// uuidKey is the space_settings key that ties a space database to its
// registry row, so a file can be recognised when it is restored or imported.
const uuidKey = "space_uuid"

// Stamp is the identity mark inside a space database.
type Stamp struct{ DB store.DB }

// SetUUID writes the uuid of the registry row this database belongs to.
func (r Stamp) SetUUID(ctx context.Context, uuid string) error {
	return db.Q(r.DB).UpsertSpaceSetting(ctx, sqlc.UpsertSpaceSettingParams{Key: uuidKey, Value: db.NS(uuid)})
}
