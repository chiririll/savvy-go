package legacy

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
)

// RolesTable keeps each user's Laravel role in a converted file. The Go
// schema allows only admin, user and guest on the server, so the conversion
// makes read-write and read-only users plain users; whoever splits the file
// reads this table to give each of them the matching role in the first space.
// It is not part of the Go schema, so nothing copies it further.
const RolesTable = "laravel_user_roles"

// mapRoles records the Laravel roles in RolesTable and maps them to the
// server roles: read-write and read-only become user.
func mapRoles(ctx context.Context, db *sql.DB) error {
	for _, q := range []string{
		fmt.Sprintf(`CREATE TABLE %s (user_id INTEGER PRIMARY KEY, role TEXT NOT NULL) STRICT`, RolesTable),
		fmt.Sprintf(`INSERT INTO %s (user_id, role) SELECT id, role FROM users`, RolesTable),
		`UPDATE users SET role = 'user' WHERE role IN ('read-write', 'read-only')`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	cols, err := columns(ctx, db, "identity_providers")
	if err != nil {
		return err
	}
	if slices.Contains(cols, "default_role") {
		if _, err := db.ExecContext(ctx,
			`UPDATE identity_providers SET default_role = 'user' WHERE default_role IN ('read-write', 'read-only')`); err != nil {
			return err
		}
	}
	return nil
}
