# 03 — Show "unconfirmed" in the users list only when mail is configured

## Problem

Every user created without a verified address gets an "Unconfirmed" badge in
the admin users list. On a server with no mailer nobody can confirm an
address, so the badge is permanent noise and suggests something is wrong.

## Where

- `resources/ts/components/features/users/columns.tsx` `createUserColumns`:
  `!row.original.emailVerified && <Badge>…emailUnverified…</Badge>`, and the
  "mark verified / unverified" menu item further down.
- `resources/ts/pages/admin/users.tsx`: already has `useMailStatus()` and
  computes `canEmail = mail.enabled && mail.appUrlSet && password_login_enabled`
  and `mustEmail = settings.require_email_verification`.
- `createUserColumns` takes seven positional parameters already (`params`
  rule soft limit is 5): this is a good moment to change it to an options
  object.

## Fix

1. Decide the rule: show the badge when `mail.enabled` **or**
   `require_email_verification` is on. (With verification required but mail
   off the admin still needs to see who is blocked; the System page already
   refuses to switch that on without mail, so this is mostly a safety net.)
2. Pass a `showUnverified: boolean` option to `createUserColumns` and gate the
   badge on it. Keep the "mark verified" menu item under the same flag, since
   it has no meaning without mail.
3. Convert the trailing parameters of `createUserColumns` to an options
   object while touching it; update `pages/admin/users.tsx` and its tests.

## Tests (write first)

- `columns` test: mail off + unverified user → no badge; mail on → badge.
- Mail off + `require_email_verification` → badge.
- Menu item "mark verified" follows the same flag.

## Done when

A mail-less server's users list has no "Unconfirmed" badges.
