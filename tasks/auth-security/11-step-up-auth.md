# 11 — Re-authenticate before credential changes

**Severity:** Low (raised by task 03 until that one is done)

## Problem

A session is enough to:

- register a passkey (`/auth/webauthn/register/*`),
- delete a passkey,
- enable 2FA (and re-enable it while pending, which resets the secret),
- issue or widen API tokens.

Someone who steals a session (XSS, an unattended device) can add their own
passkey or token and keep access after the victim changes their password.

## Where

- `internal/httpserver/auth/webauthn/webauthn.go` credential routes
- `internal/httpserver/auth/twofactor/twofactor.go` `twoFactorEnable`
- API token routes (search for `APITokens.Issue` / `UpdateGrant` callers)

## Fix

- Add a "recently authenticated" timestamp on the session
  (`reauthenticated_at`), set on login and by a new
  `POST /auth/confirm` (password, or a passkey/TOTP for SSO-only users).
- Add `guard.RequireRecentAuth(window)` (e.g. 10 minutes) and put it on the
  routes above. Answer 403 with a code like `reauth_required`, so the SPA
  can show the confirm dialog and retry.
- SPA: handle `reauth_required` in the API client with a modal.

## Tests (write first)

- Registering a passkey on a session older than the window gets 403
  `reauth_required`.
- After `POST /auth/confirm` with the right password, it succeeds.
- A wrong password on `/auth/confirm` counts toward the task 02 limits.

## Docs

Mention in `docs/` that sensitive settings ask for the password again.

## Done when

A stolen session alone can't add a lasting credential to the account.
