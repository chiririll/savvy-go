-- name: ListSchemaMigrations :many
SELECT version FROM schema_migrations;
