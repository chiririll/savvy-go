package spaces

import (
	"context"
	"errors"
	"fmt"
	"time"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/spaces"
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

func (s Service) settings() settings.Store { return settings.Store{DB: s.Store.Server()} }

// checkAdminLimit refuses making u admin of one more space when that would
// exceed max_spaces_per_user (P10). Server admins have no limit; unset means
// unlimited, 0 means no space at all.
func (s Service) checkAdminLimit(ctx context.Context, u *auth.User) error {
	if u.IsAdmin() {
		return nil
	}
	raw := s.settings().Get(ctx, "max_spaces_per_user", nil)
	limit, ok := raw.(float64)
	if !ok {
		return nil
	}
	n, err := s.registry().CountAdminSpaces(ctx, u.ID)
	if err != nil {
		return err
	}
	if float64(n) >= limit {
		return ErrSpaceLimit
	}
	return nil
}

// CreateOwned creates a space administered by owner, within their limit.
func (s Service) CreateOwned(ctx context.Context, name string, owner *auth.User) (*Space, error) {
	if owner.IsGuest() {
		return nil, ErrGuestCannotAdmin
	}
	if err := s.checkAdminLimit(ctx, owner); err != nil {
		return nil, err
	}
	return s.Create(ctx, name, owner)
}

// Rename renames a space.
func (s Service) Rename(ctx context.Context, id int64, name string) error {
	return s.registry().Rename(ctx, id, name, time.Now().UTC())
}

// Members lists a space's members.
func (s Service) Members(ctx context.Context, spaceID int64) ([]Member, error) {
	rows, err := s.registry().Members(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, len(rows))
	for i, r := range rows {
		out[i] = Member{UserID: r.UserID, Name: r.Name, Email: r.Email, Role: r.Role, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// keepsAnAdmin reports whether the space still has an admin once userID
// stops being one.
func keepsAnAdmin(ctx context.Context, reg repo.Repo, spaceID, userID int64) (bool, error) {
	role, err := reg.MemberRole(ctx, spaceID, userID)
	if err != nil {
		return false, err
	}
	if role != RoleAdmin { // not a member, or not an admin: nothing is lost
		return true, nil
	}
	admins, err := reg.CountWithRole(ctx, spaceID, RoleAdmin)
	return admins > 1, err
}

// ChangeRole changes a member's role, as a space admin does. The space keeps
// an admin (P11); a guest is never admin (P8); a new admin stays within their
// limit (P10).
func (s Service) ChangeRole(ctx context.Context, spaceID int64, target *auth.User, role string) error {
	if !ValidRole(role) {
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
	if role == RoleAdmin {
		if target.IsGuest() {
			return ErrGuestCannotAdmin
		}
		if err := s.checkAdminLimit(ctx, target); err != nil {
			return err
		}
	}
	return store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		reg := repo.Repo{DB: tx}
		if ok, err := keepsAnAdmin(ctx, reg, spaceID, target.ID); err != nil {
			return err
		} else if !ok && role != RoleAdmin {
			return ErrLastAdmin
		}
		return reg.UpsertMember(ctx, spaceID, target.ID, role, time.Now().UTC())
	})
}

// RemoveMember takes a member out of a space (also how a member leaves).
func (s Service) RemoveMember(ctx context.Context, spaceID, userID int64) error {
	return store.Tx(ctx, s.Store.Server(), func(tx store.DB) error {
		reg := repo.Repo{DB: tx}
		role, err := reg.MemberRole(ctx, spaceID, userID)
		if err != nil {
			return err
		}
		if role == "" {
			return ErrNotMember
		}
		members, err := reg.CountMembers(ctx, spaceID)
		if err != nil {
			return err
		}
		if members == 1 {
			return ErrLastMember
		}
		if role == RoleAdmin {
			if ok, err := keepsAnAdmin(ctx, reg, spaceID, userID); err != nil {
				return err
			} else if !ok {
				return ErrLastAdmin
			}
		}
		return reg.DeleteMember(ctx, spaceID, userID)
	})
}

// AssignAdmin makes target admin of a space on a server admin's behalf,
// without the space admins' consent and outside the limit. It is recorded in
// the audit log the space's admins read.
func (s Service) AssignAdmin(ctx context.Context, actor *auth.User, spaceID int64, target *auth.User) error {
	if target.IsGuest() {
		return ErrGuestCannotAdmin
	}
	if err := s.SetMember(ctx, spaceID, target, RoleAdmin); err != nil {
		return err
	}
	return s.Audit(ctx, actor, "assign_admin", &spaceID, &target.ID, target.Email)
}

// FinalBackuper keeps a last backup of a space about to be deleted.
type FinalBackuper interface {
	FinalBackup(ctx context.Context, sp Space) error
}

// Delete removes a space: a final backup is kept for server admins first.
func (s Service) Delete(ctx context.Context, sp Space, backups FinalBackuper) error {
	if err := backups.FinalBackup(ctx, sp); err != nil {
		return fmt.Errorf("final backup: %w", err)
	}
	if err := s.Store.DeleteSpace(ctx, sp.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	return s.registry().Delete(ctx, sp.ID)
}

// DeleteUser tombstones a user. Where they are the last admin of a space with
// other members the deletion is refused (P11); spaces they are alone in are
// deleted with them.
func (s Service) DeleteUser(ctx context.Context, users auth.Users, u *auth.User, backups FinalBackuper) error {
	ids, err := s.registry().AdminSpaceIDs(ctx, u.ID)
	if err != nil {
		return err
	}
	var alone []int64
	for _, id := range ids {
		members, err := s.registry().CountMembers(ctx, id)
		if err != nil {
			return err
		}
		if members == 1 {
			alone = append(alone, id)
			continue
		}
		if ok, err := keepsAnAdmin(ctx, s.registry(), id, u.ID); err != nil {
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
func (s Service) SetRoleAllowed(ctx context.Context, u *auth.User, role string) error {
	if role != auth.RoleGuest || u.IsGuest() {
		return nil
	}
	n, err := s.registry().CountAdminSpaces(ctx, u.ID)
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
func (s Service) Quota(ctx context.Context, id int64) int64 {
	return Quota(ctx, s.Store.Server(), id)
}

// Quota reads a space's size limit from the server database.
func Quota(ctx context.Context, server store.DB, id int64) int64 {
	if own, err := (repo.Repo{DB: server}).Quota(ctx, id); err == nil && own != nil {
		return *own
	}
	return settings.SpaceQuota(ctx, server)
}

// SetQuota sets or clears (bytes <= 0) a space's own size limit.
func (s Service) SetQuota(ctx context.Context, actor *auth.User, id, bytes int64) error {
	var own *int64
	if bytes > 0 {
		own = &bytes
	}
	if err := s.registry().SetQuota(ctx, id, own, time.Now().UTC()); err != nil {
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
func (s Service) Overview(ctx context.Context) ([]Overview, error) {
	rows, err := s.registry().Overview(ctx)
	if err != nil {
		return nil, err
	}
	broken := s.Store.Status().Unavailable
	out := make([]Overview, len(rows))
	for i, r := range rows {
		o := Overview{ID: r.ID, UUID: r.UUID, Name: r.Name, Members: r.Members, Admins: r.Admins,
			Unavailable: broken[r.ID], CreatedAt: r.CreatedAt}
		o.Quota = s.Quota(ctx, r.ID)
		var err error
		if o.Size, err = s.Store.SpaceSize(ctx, r.ID); errors.Is(err, store.ErrNotFound) {
			o.Unavailable = "the database file is missing"
		}
		out[i] = o
	}
	return out, nil
}

// Reconcile compares the registry with the databases the store holds (P16):
// registered spaces without a database and databases without a registered
// space. Neither is touched; both are reported for an admin to sort out.
func (s Service) Reconcile(ctx context.Context) (missing, orphans []int64, err error) {
	registered, err := s.IDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	held, err := s.Store.Spaces(ctx)
	if err != nil {
		return nil, nil, err
	}
	have := map[int64]bool{}
	for _, id := range held {
		have[id] = true
	}
	known := map[int64]bool{}
	for _, id := range registered {
		known[id] = true
		if !have[id] {
			missing = append(missing, id)
		}
	}
	for _, id := range held {
		if !known[id] {
			orphans = append(orphans, id)
		}
	}
	return missing, orphans, nil
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
func (s Service) Audit(ctx context.Context, actor *auth.User, action string, spaceID, target *int64, details string) error {
	return s.registry().InsertAudit(ctx, ownerID(actor), spaceID, target, action, details, time.Now().UTC())
}

// AuditLog lists the server admin actions on a space, newest first.
func (s Service) AuditLog(ctx context.Context, spaceID int64) ([]AuditEntry, error) {
	rows, err := s.registry().Audit(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]AuditEntry, len(rows))
	for i, r := range rows {
		out[i] = AuditEntry{ID: r.ID, Actor: r.Actor, Action: r.Action, TargetUserID: r.TargetUserID,
			Details: r.Details, CreatedAt: r.CreatedAt}
	}
	return out, nil
}
