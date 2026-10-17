# Auth security tasks

Findings from a security review of `internal/auth`, its repo layer
(`internal/db/repo/auth`) and the handlers that call it
(`internal/httpserver/auth/*`, `httpx/guard`, `httpx/login`).

Each file is one task: problem, where it is, the fix, the tests to write
first, and when it is done. Work in the order below: 01, 03 and 05 together
are what make the second factor actually hold.

| # | Task | Severity |
|---|------|----------|
| 01 | [Limit 2FA attempts per challenge](01-2fa-attempt-limit.md) | High |
| 02 | [Rate-limit public auth routes](02-auth-rate-limiting.md) | High |
| 03 | [Require user verification for passkeys](03-passkey-user-verification.md) | Medium |
| 04 | [Persist passkey sign counter, reject clones](04-passkey-sign-counter.md) | Medium |
| 05 | [Make one-time consumes atomic](05-atomic-consume.md) | Medium |
| 06 | [Reject replayed TOTP codes](06-totp-replay.md) | Medium |
| 07 | [Constant-time login for unknown emails](07-login-timing.md) | Medium |
| 08 | [Trust only the `__Host-` cookie over HTTPS](08-host-cookie-only.md) | Medium |
| 09 | [Absolute cap for remember-me sessions](09-remember-me-cap.md) | Low |
| 10 | [Harden password links](10-password-links.md) | Low |
| 11 | [Re-authenticate before credential changes](11-step-up-auth.md) | Low |
| 12 | [Origin check on public auth POSTs](12-public-post-origin.md) | Low |
| 13 | [Small hardening items](13-small-hardening.md) | Low |

## Rules that apply to every task

- Failing test first, then the fix (`AGENTS.md`).
- Schema changes edit `internal/migrate/sql/server/0001_initial.sql` in place:
  the schema is pre-release, no migration for old Go databases. Regenerate
  sqlc after editing `internal/db/queries/server/*.sql`.
- A change users can see (new error, lockout, re-auth prompt) updates `docs/`.
