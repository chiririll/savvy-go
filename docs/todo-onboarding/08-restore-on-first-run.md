# 08 Restore a server backup on first run

Depends on 01 and 03. The riskiest task, so it goes last. Used when moving from one server to another.

## Behavior

`POST /api/onboarding/restore` (multipart: `bootstrap_password`, `file`, optional `app_key` for a Laravel database). It is the usual restore without an administrator account: the server has no users yet, so the request is authorized by the bootstrap password, not by a session.

- A Laravel file goes through `restore.Server` (needs the `app_key`, saves an ordinary backup), a backup archive through `IngestServer`; then `RestoreServer`; then the password is invalidated.
- The backup's users, settings, signing key and spaces replace the empty server. Sessions are dropped (the store is replaced), so nobody is logged in.
- The SPA then clears its cached auth status and sends the person to `/login`. The restored admin logs in normally; admin setup and server setup are skipped.
- A failed restore (wrong file, wrong key) leaves the password valid.
- The endpoint exists only while there are no users.
- Add it to `httpserver/apidocs/openapi.yaml`; show the choice on the admin setup page (task 03).

## Tests (first)

- Needs the bootstrap password; refused once a user exists.
- A Laravel file with a wrong `app_key` and a bad archive leave the password valid.
- Success loads users and the signing key and invalidates the password.
- After restore `/api/auth/status` no longer says `needs_registration`.
