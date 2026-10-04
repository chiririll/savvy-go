package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

var (
	// ErrLastAdmin keeps every space with at least one admin while it has
	// other members.
	ErrLastAdmin = errors.New("a space needs an admin: make another member admin first")
	// ErrLastMember is returned when the only member would leave a space
	// behind with nobody in it; delete the space instead.
	ErrLastMember = errors.New("you are the only member of this space: delete it instead")
	// ErrSpaceLimit is returned when a user would administer more spaces than
	// max_spaces_per_user allows.
	ErrSpaceLimit = errors.New("you have reached the number of spaces you may administer")
	// ErrNotMember is returned for a user who is not in the space.
	ErrNotMember = errors.New("not a member of this space")
)

// Member is a member of a space as its other members see them.
type Member struct {
	UserID    int64
	Name      string
	Email     string
	Role      string
	CreatedAt *time.Time
}

func (s Spaces) settings() settings.Store { return settings.Store{DB: s.Store.Server()} }

// checkAdminLimit refuses making u admin of one more space when that would
// exceed max_spaces_per_user (P10). Server admins have no limit; unset means
// unlimited, 0 means no space at all.
func (s Spaces) checkAdminLimit(ctx context.Context, u *auth.User) error {
	if u.IsAdmin() {
		return nil
	}
	raw := s.settings().Get(ctx, "max_spaces_per_user", nil)
	limit, ok := raw.(float64)
	if !ok {
		return nil
	}
	n, err := s.server().CountUserAdminSpaces(ctx, u.ID)
	if err != nil {
		return err
	}
	if float64(n) >= limit {
		return ErrSpaceLimit
	}
	return nil
}

// CreateOwned creates a space administered by owner, within their limit.
func (s Spaces) CreateOwned(ctx context.Context, name string, owner *auth.User) (*Space, error) {
	if owner.IsGuest() {
		return nil, ErrGuestCannotAdmin
	}
	if err := s.checkAdminLimit(ctx, owner); err != nil {
		return nil, err
	}
	return s.Create(ctx, name, owner)
}

// Rename renames a space.
func (s Spaces) Rename(ctx context.Context, id int64, name string) error {
	return s.server().RenameSpace(ctx, sqlc.RenameSpaceParams{Name: name, UpdatedAt: db.NS(now()), ID: id})
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// Members lists a space's members.
func (s Spaces) Members(ctx context.Context, spaceID int64) ([]Member, error) {
	rows, err := s.server().ListMembers(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, r := range rows {
		out[i] = Member{UserID: r.UserID, Name: r.Name, Email: r.Email, Role: r.Role}
		if t, ok := parseNullTime(r.CreatedAt); ok {
			out[i].CreatedAt = &t
		}
	}
	return out, nil
}

// keepsAnAdmin reports whether the space still has an admin once userID
// stops being one.
func (s Spaces) keepsAnAdmin(ctx context.Context, q *sqlc.Queries, spaceID, userID int64) (bool, error) {
	role, err := q.GetMemberRole(ctx, sqlc.GetMemberRoleParams{SpaceID: spaceID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) || (err == nil && role != SpaceAdmin) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	admins, err := q.CountSpaceRole(ctx, sqlc.CountSpaceRoleParams{SpaceID: spaceID, Role: SpaceAdmin})
	return admins > 1, err
}

// ChangeRole changes a member's role, as a space admin does. The space keeps
// an admin (P11); a guest is never admin (P8); a new admin stays within their
// limit (P10).
func (s Spaces) ChangeRole(ctx context.Context, spaceID int64, target *auth.User, role string) error {
	if !ValidSpaceRole(role) {
		return fmt.Errorf("invalid space role %q", role)
	}
	current, err := s.Role(ctx, spaceID, target.ID)
	if err != nil {
		return err
	}
	if current == "" {
		return ErrNotMember
	}
	if current == role {
		return nil
	}
	if role == SpaceAdmin {
		if target.IsGuest() {
			return ErrGuestCannotAdmin
		}
		if err := s.checkAdminLimit(ctx, target); err != nil {
			return err
		}
	}
	return store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		q := db.Q(tx)
		if ok, err := s.keepsAnAdmin(ctx, q, spaceID, target.ID); err != nil {
			return err
		} else if !ok && role != SpaceAdmin {
			return ErrLastAdmin
		}
		return q.UpsertSpaceMember(ctx, sqlc.UpsertSpaceMemberParams{SpaceID: spaceID, UserID: target.ID, Role: role, CreatedAt: db.NS(now())})
	})
}

// RemoveMember takes a member out of a space (also how a member leaves).
func (s Spaces) RemoveMember(ctx context.Context, spaceID, userID int64) error {
	return store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		q := db.Q(tx)
		role, err := q.GetMemberRole(ctx, sqlc.GetMemberRoleParams{SpaceID: spaceID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotMember
		}
		if err != nil {
			return err
		}
		members, err := q.CountSpaceMembers(ctx, spaceID)
		if err != nil {
			return err
		}
		if members == 1 {
			return ErrLastMember
		}
		if role == SpaceAdmin {
			if ok, err := s.keepsAnAdmin(ctx, q, spaceID, userID); err != nil {
				return err
			} else if !ok {
				return ErrLastAdmin
			}
		}
		return q.DeleteSpaceMember(ctx, sqlc.DeleteSpaceMemberParams{SpaceID: spaceID, UserID: userID})
	})
}

// AssignAdmin makes target admin of a space on a server admin's behalf,
// without the space admins' consent and outside the limit. It is recorded in
// the audit log the space's admins read.
func (s Spaces) AssignAdmin(ctx context.Context, actor *auth.User, spaceID int64, target *auth.User) error {
	if target.IsGuest() {
		return ErrGuestCannotAdmin
	}
	if err := s.SetMember(ctx, spaceID, target, SpaceAdmin); err != nil {
		return err
	}
	return s.Audit(ctx, actor, "assign_admin", &spaceID, &target.ID, target.Email)
}

// Delete removes a space: a final backup is kept for server admins first.
func (s Spaces) Delete(ctx context.Context, sp Space, backups Backups) error {
	if _, err := backups.FinalBackup(ctx, sp); err != nil {
		return fmt.Errorf("final backup: %w", err)
	}
	if err := s.Store.DeleteSpace(ctx, sp.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	return s.server().DeleteSpaceRow(ctx, sp.ID)
}

// DeleteUser tombstones a user. Where they are the last admin of a space with
// other members the deletion is refused (P11); spaces they are alone in are
// deleted with them.
func (s Spaces) DeleteUser(ctx context.Context, users auth.Users, u *auth.User, backups Backups) error {
	ids, err := s.server().ListUserAdminSpaces(ctx, u.ID)
	if err != nil {
		return err
	}
	var alone []int64
	for _, id := range ids {
		members, err := s.server().CountSpaceMembers(ctx, id)
		if err != nil {
			return err
		}
		if members == 1 {
			alone = append(alone, id)
			continue
		}
		if ok, err := s.keepsAnAdmin(ctx, s.server(), id, u.ID); err != nil {
			return err
		} else if !ok {
			sp, _ := s.Get(ctx, id)
			name := fmt.Sprint(id)
			if sp != nil {
				name = sp.Name
			}
			return fmt.Errorf("%w (space %q)", ErrLastAdmin, name)
		}
	}
	for _, id := range alone {
		sp, err := s.Get(ctx, id)
		if err != nil || sp == nil {
			continue
		}
		if err := s.Delete(ctx, *sp, backups); err != nil {
			return err
		}
	}
	return users.Delete(ctx, u.ID)
}

// SetRoleAllowed checks a server role change: nobody who administers a
// space becomes a guest (P9).
func (s Spaces) SetRoleAllowed(ctx context.Context, u *auth.User, role string) error {
	if role != auth.RoleGuest || u.IsGuest() {
		return nil
	}
	n, err := s.server().CountUserAdminSpaces(ctx, u.ID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrGuestCannotAdmin
	}
	return nil
}

// Quota is a space's size limit in bytes (its own, or space_quota_mb), 0 for
// unlimited.
func (s Spaces) Quota(ctx context.Context, id int64) int64 {
	return SpaceQuota(ctx, s.Store.Server(), id)
}

// SpaceQuota reads a space's size limit from the server database.
func SpaceQuota(ctx context.Context, server store.DB, id int64) int64 {
	if q, err := db.Q(server).GetSpaceQuota(ctx, id); err == nil && q.Valid {
		return q.Int64
	}
	return settings.SpaceQuota(ctx, server)
}

// SetQuota sets or clears (bytes <= 0) a space's own size limit.
func (s Spaces) SetQuota(ctx context.Context, actor *auth.User, id, bytes int64) error {
	var v sql.NullInt64
	if bytes > 0 {
		v = db.NI(bytes)
	}
	if err := s.server().SetSpaceQuota(ctx, sqlc.SetSpaceQuotaParams{QuotaBytes: v, UpdatedAt: db.NS(now()), ID: id}); err != nil {
		return err
	}
	if err := s.Store.ApplyQuota(ctx, id); err != nil && !errors.Is(err, store.ErrUnavailable) {
		return err
	}
	return s.Audit(ctx, actor, "set_quota", &id, nil, fmt.Sprint(bytes))
}

// Overview is a space as the admin overview lists it: metadata only.
type Overview struct {
	ID          int64
	UUID        string
	Name        string
	Members     int64
	Admins      int64
	Size        int64
	Quota       int64
	Unavailable string
	CreatedAt   *time.Time
}

// Overview lists every space with its size and members, never its data (P5).
func (s Spaces) Overview(ctx context.Context) ([]Overview, error) {
	rows, err := s.server().ListSpacesOverview(ctx)
	if err != nil {
		return nil, err
	}
	broken := s.Store.Status().Unavailable
	out := make([]Overview, len(rows))
	for i, r := range rows {
		o := Overview{ID: r.ID, UUID: r.Uuid, Name: r.Name, Members: r.Members, Admins: r.Admins, Unavailable: broken[r.ID]}
		o.Quota = s.Quota(ctx, r.ID)
		o.Size, _ = s.Store.SpaceSize(ctx, r.ID)
		if t, ok := parseNullTime(r.CreatedAt); ok {
			o.CreatedAt = &t
		}
		out[i] = o
	}
	return out, nil
}

// AuditEntry is a server admin action on a space.
type AuditEntry struct {
	ID           int64
	Actor        string
	Action       string
	TargetUserID *int64
	Details      string
	CreatedAt    *time.Time
}

// Audit records a server admin action (P53).
func (s Spaces) Audit(ctx context.Context, actor *auth.User, action string, spaceID, target *int64, details string) error {
	var actorID, sid, tid sql.NullInt64
	if actor != nil {
		actorID = db.NI(actor.ID)
	}
	if spaceID != nil {
		sid = db.NI(*spaceID)
	}
	if target != nil {
		tid = db.NI(*target)
	}
	return s.server().InsertAudit(ctx, sqlc.InsertAuditParams{
		ActorID: actorID, Action: action, SpaceID: sid, TargetUserID: tid, Details: db.NullStringVal(details), CreatedAt: now(),
	})
}

// AuditLog lists the server admin actions on a space, newest first.
func (s Spaces) AuditLog(ctx context.Context, spaceID int64) ([]AuditEntry, error) {
	rows, err := s.server().ListSpaceAudit(ctx, db.NI(spaceID))
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, len(rows))
	for i, r := range rows {
		e := AuditEntry{ID: r.ID, Actor: r.Actor, Action: r.Action, Details: r.Details.String}
		if r.TargetUserID.Valid {
			id := r.TargetUserID.Int64
			e.TargetUserID = &id
		}
		if t, err := time.Parse(time.RFC3339, r.CreatedAt); err == nil {
			e.CreatedAt = &t
		}
		out[i] = e
	}
	return out, nil
}
