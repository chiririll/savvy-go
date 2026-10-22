package legacy

import (
	"context"
	"database/sql"
)

// mapPasswords rewrites Laravel's $2y$ bcrypt prefix to the $2a$ one the Go
// side writes, so nothing outside this package sees a Laravel hash.
func mapPasswords(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET password = '$2a$' || substr(password, 5) WHERE password LIKE '$2y$%'`)
	return err
}
