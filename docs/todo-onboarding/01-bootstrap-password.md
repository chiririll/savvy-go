# 01 Bootstrap password and register gate

Backend package `internal/bootstrap`. Closes the hole where the first visitor of a fresh server can claim admin.

## Rules

- Registration is never open. While there are no users, `POST /api/auth/register` and the restore endpoint (task 08) require the bootstrap password. Once a user exists, registration stays closed as today (`403`). Invitation registration is a separate path and is not touched.
- Generated in `cmd/savvy-go` after `seed.Run`, only if the user count is 0 (so a demo-seeded server gets none). 128 bits from `crypto/rand`, base32 groups for easy typing. At that size guessing is not a realistic attack, so there is no rate limiter.
- Kept in memory and written to `<data>/bootstrap-password` (`0600`). If the file cannot be written, log a warning and rely on the log alone. A restart reuses the file's value while there are no users.
- Printed once at startup in one clear log block: the password, the file path and the URL to open.
- Compared in constant time (`secureCompare`). A mutex makes verify + create + invalidate one step. This also fixes the existing race, where two simultaneous registrations on an empty server both succeed.
- Invalidated, and the file removed, as soon as the first user exists (registration or restore).

## API

- `POST /api/auth/register` gets the body field `bootstrap_password` (required while there are no users).
- `GET /api/auth/status` also returns `bootstrap_hint`: the environment (`docker`, `systemd`, `binary`; decided at start from `/.dockerenv` or `INVOCATION_ID`) and the file path.
- Update `httpserver/apidocs/openapi.yaml`.

The page that shows the field and the per-environment instructions is task 03. Instructions it needs: `docker logs <container>` or `docker exec <container> cat /data/bootstrap-password`; `journalctl -u savvy-go` or `sudo cat /var/lib/savvy-go/bootstrap-password`; the terminal or the file for a binary.

## Code

- New `internal/bootstrap`; add it to the package list in `internal/AGENTS.md` and to `businessPackages` in `store/arch_test.go`.
- `authRegister` calls it; `cmd/savvy-go` creates it after `seed.Run`.
- Until task 03 lands, add the password field to the existing `setup.tsx` so the app stays usable.

## Tests (first)

- Not created when users exist or after demo seeding.
- Reused from the file after a restart; file mode `0600`; unwritable file only logs.
- Constant-time compare; two parallel registrations create exactly one admin.
- Invalidated (and the file removed) after the first user exists.
- Register without or with a wrong password is refused; with users present it is refused (`403`).
- `bootstrap_hint` per environment.

## Done when

A fresh server prints the password, registration needs it, and the old open `/setup` flow no longer works without it.
