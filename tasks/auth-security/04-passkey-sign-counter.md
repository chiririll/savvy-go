# 04 — Persist passkey sign counter, reject clones

**Severity:** Medium

## Problem

The stored credential record keeps the sign count from registration forever.
`FinishLogin` throws away the credential that validation returns, and
`touchCred` only updates `last_used_at`. go-webauthn's clone detection
(`Authenticator.CloneWarning`) compares against the stale count, so it never
fires, and a cloned authenticator goes unnoticed.

## Where

- `internal/auth/webauthn.go` `FinishLogin`: the result of `wa.ValidateLogin`
  is discarded (`_`), and the result of `ValidateDiscoverableLogin` is used
  only for `cred.ID`.
- `touchCred` → `repo.TouchPasskey` (`TouchWebAuthnCredential` query).
- Table `webauthn_credentials` has `record` (JSON) and `counter`.

## Fix

1. Replace `TouchPasskey` with `UpdatePasskeyAfterLogin(credentialID, record,
   counter, now)`, which writes the re-marshalled `webauthn.Credential` and
   `cred.Authenticator.SignCount`.
2. In both login paths, keep the returned `*webauthn.Credential`. If
   `cred.Authenticator.CloneWarning` is true, refuse the login (`verify`
   error) and log a warning with the user and credential id. Otherwise save it.
3. Authenticators that always report 0 (many platform passkeys) don't trigger
   CloneWarning. That's fine, and the library handles it.

## Tests (write first)

- After a login, the stored counter equals the authenticator's counter.
- A replayed assertion whose counter is not higher than the stored one, from
  an authenticator with a non-zero counter, is rejected.

## Done when

The counter moves forward with each login, and an assertion that would set
off clone detection is refused.
