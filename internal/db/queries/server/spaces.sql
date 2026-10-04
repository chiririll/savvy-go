-- name: InsertSpace :one
INSERT INTO spaces (uuid, name, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: DeleteSpaceRow :exec
DELETE FROM spaces WHERE id = ?;

-- name: GetSpace :one
SELECT id, uuid, name, quota_bytes, created_by, created_at, updated_at FROM spaces WHERE id = ?;

-- name: RenameSpace :exec
UPDATE spaces SET name = ?, updated_at = ? WHERE id = ?;

-- name: SetSpaceQuota :exec
UPDATE spaces SET quota_bytes = ?, updated_at = ? WHERE id = ?;

-- name: GetSpaceQuota :one
SELECT quota_bytes FROM spaces WHERE id = ?;

-- name: ListSpacesOverview :many
SELECT s.id, s.uuid, s.name, s.quota_bytes, s.created_at,
	(SELECT COUNT(*) FROM space_members m WHERE m.space_id = s.id) AS members,
	(SELECT COUNT(*) FROM space_members m WHERE m.space_id = s.id AND m.role = 'admin') AS admins
FROM spaces s ORDER BY s.id;

-- name: ListSpaceIDs :many
SELECT id FROM spaces ORDER BY id;

-- name: ListUserSpaces :many
SELECT s.id, s.uuid, s.name, m.role
FROM space_members m JOIN spaces s ON s.id = m.space_id
WHERE m.user_id = ?
ORDER BY s.id;

-- name: GetMemberRole :one
SELECT role FROM space_members WHERE space_id = ? AND user_id = ?;

-- name: UpsertSpaceMember :exec
INSERT INTO space_members (space_id, user_id, role, created_at) VALUES (?, ?, ?, ?)
ON CONFLICT (space_id, user_id) DO UPDATE SET role = excluded.role;

-- name: SpaceIDByUUID :one
SELECT id FROM spaces WHERE uuid = ?;

-- name: ListMembers :many
SELECT m.user_id, u.name, u.email, m.role, m.created_at
FROM space_members m JOIN users u ON u.id = m.user_id
WHERE m.space_id = ?
ORDER BY u.name;

-- name: DeleteSpaceMember :exec
DELETE FROM space_members WHERE space_id = ? AND user_id = ?;

-- name: CountSpaceRole :one
SELECT COUNT(*) FROM space_members WHERE space_id = ? AND role = ?;

-- name: CountSpaceMembers :one
SELECT COUNT(*) FROM space_members WHERE space_id = ?;

-- name: CountUserAdminSpaces :one
SELECT COUNT(*) FROM space_members WHERE user_id = ? AND role = 'admin';

-- name: ListUserAdminSpaces :many
SELECT space_id FROM space_members WHERE user_id = ? AND role = 'admin' ORDER BY space_id;

-- name: DeleteUserMemberships :exec
DELETE FROM space_members WHERE user_id = ?;

-- name: InsertInvitation :one
INSERT INTO space_invitations (space_id, email, role, token_hash, invited_by, invited_by_server_admin, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetInvitationByHash :one
SELECT i.id, i.space_id, s.name AS space_name, i.email, i.role, i.invited_by_server_admin,
	i.expires_at, i.accepted_at, COALESCE(u.name, '') AS inviter
FROM space_invitations i JOIN spaces s ON s.id = i.space_id LEFT JOIN users u ON u.id = i.invited_by
WHERE i.token_hash = ?;

-- name: ListInvitations :many
SELECT i.id, i.email, i.role, i.expires_at, i.accepted_at, i.created_at, COALESCE(u.name, '') AS inviter
FROM space_invitations i LEFT JOIN users u ON u.id = i.invited_by
WHERE i.space_id = ?
ORDER BY i.id DESC;

-- name: AcceptInvitation :execresult
UPDATE space_invitations SET accepted_at = ?, accepted_by = ?
WHERE id = ? AND accepted_at IS NULL;

-- name: DeleteInvitation :execresult
DELETE FROM space_invitations WHERE id = ? AND space_id = ?;

-- name: InsertAudit :exec
INSERT INTO admin_audit (actor_id, action, space_id, target_user_id, details, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListSpaceAudit :many
SELECT a.id, a.action, a.target_user_id, a.details, a.created_at, COALESCE(u.name, '') AS actor
FROM admin_audit a LEFT JOIN users u ON u.id = a.actor_id
WHERE a.space_id = ?
ORDER BY a.id DESC
LIMIT 200;
