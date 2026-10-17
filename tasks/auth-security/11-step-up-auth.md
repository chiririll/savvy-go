# 11 — Confirm before credential changes

**Severity:** Low (raised by task 03 until that one is done)

**Status:** Done in code. Left open: SSO-only accounts, see below.

## Problem

A session was enough to:

- register a passkey (`/auth/webauthn/register/*`),
- delete a passkey,
- enable 2FA (and re-enable it while pending, which resets the secret),
- issue or widen API tokens.

Someone who steals a session (XSS, an unattended device) could add their own
passkey or token and keep access after the victim changes their password.

## Fix

No session state. The request that makes the change also carries the proof, so
there is no timestamp to go stale and no "confirm" step to skip.

- `httpx/confirm.Confirmer.Require` reads `current_password` and `code` from
  the body. With 2FA on it wants `code` (authenticator or recovery); otherwise
  the password; an account with neither (SSO only) passes, it has nothing to
  confirm with.
- Missing: 403 `confirmation_required`. Wrong: 422 `confirmation_invalid`.
  Every attempt counts against the sign-in account limit (task 02).
- Applies to: passkey register `options` (`finish` only completes a challenge
  issued after the check), passkey delete, 2FA enable, token create, token
  update and approving an AI app on the OAuth consent screen (not declining). On the token routes it runs after validation, so a bad form costs no
  attempt.
- Revoking a token and renaming a passkey take nothing away from the owner and
  ask for nothing. Change password, disable 2FA and regenerate codes already
  required the current secret.
- SPA: `withConfirmation` (`stores/confirm.ts`) opens a cancellable
  "Authentication required" dialog, sends the change with what was typed, keeps
  the dialog open on a wrong answer, and rejects with `ConfirmCancelled` when
  closed.

## Left open

- An SSO-only account with no 2FA can be given a 2FA secret by a stolen session.
  Nothing to confirm with; it is the price of not locking those users out.

## Tests

`internal/httpserver/confirm_test.go`, `resources/ts/stores/confirm.test.ts`.

## Docs

`docs/api.md`, and the token routes in `openapi.yaml`.
