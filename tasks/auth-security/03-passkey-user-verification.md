# 03 — Require user verification for passkeys

**Severity:** Medium

## Problem

A passkey login is a full sign-in: it skips both the password and TOTP. But
the ceremony never asks for user verification (a PIN or biometric).
go-webauthn v0.18 defaults to `preferred`, and it only enforces the UV flag
when the session says `required`. So whoever holds a security key with no
PIN can sign in as its owner with just a tap.

## Where

`internal/auth/webauthn.go`:

- `BeginLogin`: `wa.BeginDiscoverableLogin()` and `wa.BeginLogin(...)` are
  called with no options.
- `BeginRegistration`: no `AuthenticatorSelection.UserVerification`.

## Fix

- Login: pass `webauthn.WithUserVerification(protocol.VerificationRequired)`
  to both `BeginDiscoverableLogin` and `BeginLogin`.
- Registration: set
  `webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired})`,
  replacing the separate `WithResidentKeyRequirement`.
- Optional: the session stores the requirement, so `ValidateLogin` /
  `ValidateDiscoverableLogin` then reject assertions without the UV flag.
  Add a test that proves it.

## Tests (write first)

`webauthn_flow_test.go` already drives a virtual authenticator:

- An assertion with the UV flag cleared is rejected (discoverable and
  user-bound flows).
- An assertion with UV set still signs in.
- The registration options JSON has `userVerification: "required"`.

## Docs

Mention in `docs/` that a passkey needs a PIN or biometric. Keys without
one can no longer be registered.

## Done when

No passkey login succeeds without user verification.
