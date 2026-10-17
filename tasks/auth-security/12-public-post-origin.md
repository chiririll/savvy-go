# 12 — Origin check on public auth POSTs

**Severity:** Low

## Problem

CSRF protection (`guard.RequireCSRF`) only covers routes behind a session.
The public POSTs (`/auth/login`, `/auth/2fa/verify`,
`/auth/webauthn/login/verify`, `/auth/password/{token}`) decode JSON without
checking `Content-Type` or `Origin`. A cross-site `<form enctype="text/plain">`
can send a JSON-shaped body. `SameSite=Lax` doesn't stop this, because no
cookie is needed. That allows login CSRF: the victim is signed into the
attacker's account.

## Where

- `internal/httpserver/httpx/routing` `Build` (`mount(public, r)`)
- The public handlers listed above

## Fix

- Add middleware for unsafe methods on `/api` public routes:
  - require `Content-Type: application/json` (this alone stops simple form
    posts), and
  - when `Origin` (or `Sec-Fetch-Site`) is present, require it to match
    `app_url`'s origin, or `same-origin`.
- Leave out the routes that must accept cross-origin requests on purpose
  (OAuth `/token`, `/register`, MCP), which have their own rules.

## Tests (write first)

- POST `/auth/login` with `Content-Type: text/plain` → 415.
- POST with `Origin: https://evil.example` → 403.
- Same-origin JSON POST → passes.
- OAuth token endpoint still accepts its cross-origin requests.

## Done when

No cross-site page can drive a sign-in.
