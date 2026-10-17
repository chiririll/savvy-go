# 09 — Absolute cap for remember-me sessions

**Severity:** Low

## Problem

`Sessions.Refresh` sets `AbsoluteExpiresAt = now + RememberTTL` on every
daily rotation. A remember-me session therefore never ends while it's used
at least once per `RememberTTL` (7 days by default). A stolen remember-me
cookie works indefinitely.

## Where

`internal/auth/session.go` `Refresh` and `RotateSession` in the repo
(`RefreshSession` query).

## Fix

- Add `max_session_age` to `[security]` (minutes, default e.g. 30 days).
- In `Refresh`, cap the new expiry at `CreatedAt + MaxSessionAge`. Once that
  passes, `Resolve` returns nil and the user signs in again.
- Rotation keeps the absolute expiry it already has. Only the idle expiry
  slides.

## Tests (write first)

- With the clock moved past `CreatedAt + MaxSessionAge` and daily refreshes
  in between, `Resolve` returns nil.
- `RefreshDueAt` never goes past the cap.

The package uses `time.Now()` directly, so this may need an injectable clock
on `Sessions`.

## Docs

Document `max_session_age` in `docs/deployment.md`.

## Done when

Every session has a hard end counted from when it was created.
