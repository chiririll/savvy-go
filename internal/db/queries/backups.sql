-- name: ListBackups :many
SELECT id, filename, size, note, app_version, schema_migrations, created_at
FROM backups ORDER BY created_at DESC, id DESC;

-- name: GetBackup :one
SELECT id, filename, size, note, app_version, schema_migrations, created_at
FROM backups WHERE id = ?;

-- name: InsertBackup :execresult
INSERT INTO backups (filename, size, note, app_version, created_at, updated_at) VALUES (?,?,?,?,?,?);

-- name: DeleteBackup :exec
DELETE FROM backups WHERE id = ?;

-- name: ListSchemaMigrations :many
SELECT version FROM schema_migrations;
