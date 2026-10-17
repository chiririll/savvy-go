# 07 — Constant-time login for unknown emails

**Severity:** Medium

## Problem

`authLogin` returns at once when no user has the email (and for SSO-only or
inactive users). For a real user it runs bcrypt first. The difference, about
50 ms at cost 10, shows which emails have accounts.

## Where

`internal/httpserver/auth/session/session.go` `authLogin`, the condition
chain at the `ByEmail` call.

## Fix

- Add `password.Dummy()` (or `password.CheckOrBurn(hash, plain)`), which
  runs `bcrypt.CompareHashAndPassword` against a fixed hash made once at
  startup. It should take the same time as a real check.
- In `authLogin`, always run exactly one bcrypt compare, with the dummy when
  the user is missing, SSO-only or inactive, and then build the result.
- Check other public paths for the same leak. `passwordPreview` reveals only
  to whoever holds a token, which is fine. Check that any "resend
  verification" or "forgot password" route answers the same way for unknown
  emails.

## Tests (write first)

- `password`: the dummy check returns false for any input.
- Handler: an unknown email and a wrong password give identical bodies and
  status codes. A timing test is flaky, so instead assert that the dummy path
  is called, by injecting the checker.

## Done when

The login response doesn't depend on whether the email exists, in body,
status or the bcrypt work done.
