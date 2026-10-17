# 13 — Small hardening items

**Severity:** Low. Each item is small; do them together or alongside related tasks.

- [x] **Modulo bias in random strings.** `RandomString` (`internal/auth/secret.go`)
  and `recoveryCode` (`internal/auth/twofactor.go`) use `byte % len(alphabet)`
  (256 % 62 and 256 % 31). The loss is a few bits out of 280+, so it's
  harmless in practice, but rejection sampling (drop bytes ≥ 248 / 248) makes
  the output uniform. Also, `recoveryCode` ignores the `rand.Read` error:
  panic like `RandomString` does.
  *Test:* a chi-square-style check over many characters stays within bounds.

- [x] **Constant-time TOTP compare.** `totp.Verify` uses `==`. Switch to
  `subtle.ConstantTimeCompare` (also listed in task 06).

- [x] **`Users.Update` doesn't check the role.** It writes any `role`
  string. Return an error unless `ValidRole(NormalizeRole(role))`, so the
  rule doesn't depend on every caller.
  *Test:* `Update` with `role="root"` fails.

- [x] **Errors that are ignored.**
  - `TwoFactor.Regenerate` ignores the `InvalidateUnusedRecoveryCodes`
    error. If it fails, the old codes keep working alongside the new ones.
    Return it, and run invalidate + generate in one `store.Tx`.
  - `TwoFactor.Disable` ignores the `DeleteRecoveryCodes` error. Put it in
    the same transaction as `SetTwoFactor`.
  - `TwoFactor.Confirm` / `generateCodes`: insert the 8 codes in one
    transaction, so a failure doesn't leave some of them behind.
  - `consumeChallengeAny` / `TakePasskeyChallenge` hide DB errors (see task 05).

- [x] **Re-enabling 2FA while pending.** `Enable` resets the secret
  whenever 2FA isn't confirmed. That's fine, but it should also clear any
  stale recovery codes, and `Confirm` should check that recovery codes are
  only created once.

- [x] **Unused code in `webauthn.go`.** `init() { _ = rand.Reader }` does
  nothing; remove it along with the `crypto/rand` import. In
  `webauthnLoginOptions`, `userID` / `u2` are dead too.

## Done when

Every box is ticked, and each item has a test where the item says so.
