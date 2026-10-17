# 08 — Trust only the `__Host-` cookie over HTTPS

**Severity:** Medium

## Problem

1. **Cookie tossing.** `Security.ReadToken` falls back to the plain
   `svy_session` cookie when there is no `__Host-svy_session`, even on
   HTTPS. A sibling subdomain, or anyone who can change a plain-HTTP
   response, can plant `svy_session`. That fixes the victim's session to
   the attacker's account (login CSRF), so data the victim enters goes to
   the attacker.
2. **Client-controlled `Secure`.** `IsSecure` trusts `X-Forwarded-Proto`
   from any client. When the server is reachable without a proxy, the client
   chooses the cookie flags and name.

## Where

`internal/auth/cookies.go` `ReadToken`, `IsSecure`.

## Fix

1. Have `ReadToken` take the `secure` decision: over HTTPS, read only
   `__Host-` + name, and over HTTP, read only the plain name.
2. Decide `secure` from config, not headers. If `app_url` is `https://...`,
   it's secure. Without `app_url`, use `r.TLS != nil`. Trust
   `X-Forwarded-Proto` only behind a setting naming trusted proxies, if
   that's needed at all.
3. Update the callers: `guard.RequireSession`, `session.authMe`, logout, and
   anywhere else that calls `ReadToken`.

## Tests (write first)

- HTTPS request with only `svy_session` set: not authenticated.
- HTTPS request with `__Host-svy_session`: authenticated.
- HTTP request with `X-Forwarded-Proto: https` and an `http://` app_url: the
  cookie is not `Secure`, and the name has no prefix.

## Docs

`docs/deployment.md`: explain how HTTPS is detected (from `app_url`).

## Done when

Over HTTPS, only a `__Host-` cookie can authenticate, and request headers
alone can't change the cookie flags.
