# 03 Admin setup page (flow 1)

Depends on 01. The `/setup` page moves into the onboarding folder and is rewritten as two screens.

1. **Bootstrap password**, with the instructions for the environment (`bootstrap_hint` from `GET /api/auth/status`, see task 01). Two choices: **Create admin**, or **Restore server backup** (task 08; hide the choice until it exists).
2. **Create admin**: name, email, password (as today). The request carries the bootstrap password. No personal space is created. Login is automatic, then the SPA goes to server setup (task 04).

## Work

- New `pages/onboarding/` and `components/features/onboarding/` (layout, `api/onboarding.ts`).
- Remove `pages/auth/setup.tsx`; point the `setup` route in `app/pages.ts` and `app/router.tsx` at the new page.
- `login.tsx` and `AuthProvider` keep redirecting on `needs_registration`; the status cache is dropped after registration, as today.
- Strings in `locales/{en,ru}`.

## Tests (first)

- Wrong password shows an error and does not create a user.
- The instructions match each environment hint (docker, systemd, binary).
- After a successful registration the user is logged in and sent to server setup.
- Locale keys exist in `en` and `ru`.
