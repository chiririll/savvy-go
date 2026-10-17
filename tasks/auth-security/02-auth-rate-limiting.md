# 02 — Rate-limit public auth routes

**Severity:** High

## Problem

None of the unauthenticated auth endpoints are throttled, so:

- Passwords can be guessed online without limit. The first admin's password
  only has to be 6 characters (`authRegister`), while change and reset require 8.
- Each 2FA guess can run up to 8 bcrypt compares against recovery codes
  (`TwoFactor.ConsumeRecovery`), about 0.5 s of CPU. Unauthenticated callers
  can use that to tie up the server.

## Where

- `internal/httpserver/httpx/routing` `Build`: `mount(public, r)` has no
  middleware.
- Routes: `/auth/login`, `/auth/2fa/verify`, `/auth/bootstrap/verify`,
  `/auth/register`, `/auth/password/{token}` (GET and POST),
  `/auth/webauthn/login/*`, `/auth/verify-email/*`.
- `internal/auth/twofactor.go` `ConsumeRecovery`, `generateCodes`.
- `internal/httpserver/auth/session/session.go:150` (6-char minimum).

## Fix

1. Add an in-memory token-bucket limiter (one Go process, so no Redis), keyed
   by client IP. `RealIP` already puts the client address in `RemoteAddr`.
   Add a second key, the normalised email, for `/auth/login`. Answer 429
   with `Retry-After`. Suggested starting limits: 10 per minute per IP, 5 per
   minute per account. Make them configurable in `[security]`.
2. Put the limiter on the routes above, either as a group in `routing.Build`
   or per route.
3. Store recovery codes as HMAC-SHA256 (key from the server secret) or plain
   SHA-256 instead of bcrypt. They have about 40 random bits and are never
   chosen by a user, so a slow hash adds nothing. Look the code up by hash
   instead of looping over every row. This needs a schema change: index on
   `(user_id, code)`.
4. Require 8 characters in `authRegister` too, and move the minimum into one
   constant used by every password path.

## Tests (write first)

- Limiter unit tests: a burst over the limit gets 429, and requests pass
  again after the window.
- Handler test: the 11th login in a minute from one IP gets 429.
- Recovery code: an old bcrypt row (if any exist) is not required, since
  the schema is pre-release. A new code verifies once and fails the second time.
- Register with a 7-character password fails validation.

## Docs

Document the limits and the `[security]` keys in `docs/deployment.md`.

## Done when

No public auth route takes unlimited attempts, and a 2FA guess no longer
costs bcrypt × 8.
