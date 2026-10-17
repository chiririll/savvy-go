# 05 — Make one-time consumes atomic

**Severity:** Medium

## Problem

Several "use once" credentials can be used twice by two requests that
arrive together:

1. **Recovery codes.** `MarkRecoveryCodeUsed` is
   `UPDATE ... SET used_at=? WHERE id=?`. It doesn't check `used_at IS NULL`
   or the affected-row count. Two requests can both accept the same code.
2. **2FA challenge.** In `twoFactorVerify`, `Challenges.Consume` returns
   `(nil, nil)` when the challenge was already consumed, and the handler only
   checks `err`. Two requests at once with one challenge and a valid code
   both get a session.
3. **Passkey challenge.** `TakePasskeyChallenge` runs a SELECT, then an
   UPDATE whose result it ignores, and it hides SELECT errors as "not found".
4. **Passkey + 2FA token.** `webauthnLoginVerify` ignores the result of
   `Challenges.Consume`.

## Where

- `internal/db/queries/server/recovery.sql` `MarkRecoveryCodeUsed`
- `internal/db/repo/auth/auth.go` `MarkRecoveryCodeUsed`, `TakePasskeyChallenge`
- `internal/db/queries/server/webauthn.sql` `ConsumeWebAuthnChallenge`
- `internal/auth/twofactor.go` `ConsumeRecovery`
- `internal/httpserver/auth/twofactor/twofactor.go` `twoFactorVerify`
- `internal/httpserver/auth/webauthn/webauthn.go` `webauthnLoginVerify`

## Fix

1. `MarkRecoveryCodeUsed`: `WHERE id=? AND used_at IS NULL`, return
   `(bool, error)` from rows affected. `ConsumeRecovery` returns true only
   when this call is the one that marked the code.
2. `twoFactorVerify`: treat `u == nil` from `Consume` as "Invalid or expired
   token." (401). Better still, consume first and verify the code after, so
   the challenge is claimed atomically (pairs with task 01's attempt counter).
3. `TakePasskeyChallenge`: one statement,
   `UPDATE webauthn_challenges SET consumed_at=? WHERE token_hash=? AND type=? AND consumed_at IS NULL AND expires_at > ? RETURNING user_id, options`
   (SQLite ≥ 3.35). Return real DB errors.
4. `webauthnLoginVerify`: if consuming the 2FA token fails, refuse the login.

## Tests (write first)

- Recovery code: two `ConsumeRecovery` calls with the same code, the second
  returns false. Add a concurrent version with goroutines against a real
  SQLite file.
- 2FA verify: consume the challenge between `Resolve` and `Consume` (or call
  the handler twice), and the second call gets no session.
- Passkey challenge: `TakePasskeyChallenge` twice on the same hash, the
  second returns nil.

## Done when

Each one-time token or code can produce at most one success, even when
requests race.
