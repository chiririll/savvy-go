# Auth

- Root `auth` owns the user model: users, sessions, API tokens, 2FA, passkeys, one-time tokens.
- New pure helper with no model dependency: new subpackage, not a file in root. No aliases left behind when moving a symbol.
- New API resource: add it to the preset matrix in `permissions/permissions_test.go` so every preset places it on purpose.
