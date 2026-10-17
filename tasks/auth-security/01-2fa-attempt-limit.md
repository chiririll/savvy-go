# 01 — Limit 2FA attempts per challenge

**Severity:** High

## Problem

`POST /api/auth/2fa/verify` can be brute-forced. A wrong code returns 422
and leaves the challenge open, so the caller can keep guessing until it
expires (5 minutes by default). When it expires, logging in again with the
password issues a new one. Each guess has a 3 in 10⁶ chance, because TOTP
accepts the steps either side of the current one. With nothing counting
failures, the second factor only slows an attacker who has the password.

## Where

- `internal/httpserver/auth/twofactor/twofactor.go` `twoFactorVerify`: calls
  `Challenges.Resolve` (no consume), then `VerifyAny`; a failure returns 422
  and the challenge stays valid.
- `internal/auth/challenge.go`: no attempt counter.
- `internal/httpserver/auth/webauthn/webauthn.go` `webauthnLoginOptions` /
  `webauthnLoginVerify` also take the same challenge token.
- Table `two_factor_challenges` in `internal/migrate/sql/server/0001_initial.sql`.

## Fix

1. Add `attempts INTEGER NOT NULL DEFAULT 0` to `two_factor_challenges`.
2. Add `Challenges.Fail(ctx, token) (remaining int, err error)`: in one
   statement, increment `attempts` and set `consumed_at` once
   `attempts >= maxChallengeAttempts` (5).
3. In `twoFactorVerify`, on a wrong code call `Fail`. When the challenge is
   used up, return 401 "Invalid or expired token." so the SPA goes back to
   the login form.
4. Count consecutive 2FA failures per user across challenges as well (column
   on `users`, or a small table). After N failures (e.g. 10), refuse new
   challenges for a growing delay, and reset the count on success. Without
   this, an attacker gets a fresh 5 guesses with every password login.

## Tests (write first)

- `internal/auth`: after 5 `Fail` calls, `Resolve` returns nil.
- Handler test: 5 wrong codes, then the right code returns 401, not a session.
- Handler test: the right code on the 4th try still signs in.
- Per-user backoff: after N failures across challenges, login returns the
  lockout answer, and a success resets it.

## Docs

Say in `docs/` that 2FA locks after repeated wrong codes and how long the
lockout lasts.

## Done when

You can't send more than 5 guesses per challenge, and a burst of failed
challenges makes the account wait instead of getting new ones straight away.
