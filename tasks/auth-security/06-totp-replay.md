# 06 — Reject replayed TOTP codes

**Severity:** Medium

## Problem

`totp.Verify` accepts any code in the window from one step before to one
step after, and nothing remembers which step was last used. A code someone
has seen (shoulder-surfed, phished, logged) stays valid for about 90 s. That
covers login, disable and regenerate.

## Where

- `internal/auth/totp/totp.go` `Verify`
- `internal/auth/twofactor.go` `verifyTOTP`, `VerifyAny`, `Confirm`, `Regenerate`
- Table `users` (no last-step column)

## Fix

1. In `totp`, add `VerifyStep(secret, code string, now time.Time) (step int64, ok bool)`,
   which returns the matched counter. Keep `totp` pure (no storage), as
   `internal/auth/AGENTS.md` requires.
2. Add `two_factor_last_step INTEGER` to `users`.
3. In `TwoFactor.verifyTOTP`, accept only when `step > last_step`, and
   update it atomically:
   `UPDATE users SET two_factor_last_step=? WHERE id=? AND (two_factor_last_step IS NULL OR two_factor_last_step < ?)`
   and check rows affected.
4. While you're there, compare codes with `subtle.ConstantTimeCompare`.

## Tests (write first)

- `totp`: `VerifyStep` returns the right step for a code from the previous
  and next step (inject `now`).
- `auth`: the same valid code works once and fails the second time.
- A code from an older step than the last accepted one fails.

## Done when

Each TOTP code is accepted at most once per user.
