# Auth

- Root `auth` owns the user model: users, sessions, API tokens, 2FA, passkeys, one-time tokens.
- Subpackages (`totp`, `password`, `permissions`) are pure leaves: no storage, never import `auth`. Reason: `auth` imports them, so the reverse is an import cycle. Shared vocabulary they need (`permissions.ErrBadGrant`, token kinds) lives in them.
- New pure helper with no model dependency: new subpackage, not a file in root. No aliases left behind when moving a symbol.
- Laravel hashes never reach this package. `legacy.Upgrade` rewrites `$2y$` to `$2a$` once.
- New API resource: add it to the preset matrix in `permissions/permissions_test.go` so every preset places it on purpose.
