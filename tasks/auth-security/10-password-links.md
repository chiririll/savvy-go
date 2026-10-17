# 10 — Harden password links

**Severity:** Low

## Problem

1. Password tokens live 7 days (`passwordTokenTTL`). That's reasonable for
   an invite to a new, inactive account, but long for a reset of an
   existing password.
2. The token sits in the URL path (`/auth/password/{token}`), so it ends up
   in proxy and access logs and can leak through `Referer`.
3. A password reset revokes sessions but not API tokens. Someone who held
   the account before the reset keeps API access.

## Where

- `internal/auth/password_token.go`
- `internal/httpserver/auth/session/session.go` `passwordPreview`, `passwordAccept`
- Whoever issues the tokens (admin users routes)

## Fix

1. Give tokens a purpose: `invite` (7 days) and `reset` (1 hour,
   configurable). Store the purpose, or pass the TTL to `Issue`.
2. Add `Referrer-Policy: no-referrer` (at least on the SPA page for this
   route and on the API answers). Make sure the request logger doesn't log
   this path in full.
3. In `passwordAccept`, revoke the user's API tokens too, or at least offer
   it. Decide and write it down. Do the same in `authChangePassword` if you
   choose "revoke".

## Tests (write first)

- A reset token expires after its TTL, and an invite token after its own.
- After `passwordAccept`, the user's API tokens no longer resolve (if you
  choose to revoke them).
- The response to `GET /auth/password/{token}` has the `Referrer-Policy` header.

## Docs

Say in `docs/` how long links work and what a reset signs out.

## Done when

Reset links are short-lived, don't leak through Referer, and a reset ends
every way into the account that existed before it.
