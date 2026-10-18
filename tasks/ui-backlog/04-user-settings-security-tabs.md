# 04 — Rename "User" to "Security", drop Preferences, give passkeys a tab

Covers two notes: rename the page and remove the Preferences tab; move
passkeys to their own tab.

## Problem

Settings → User has three tabs: Preferences, Security, API.

- Preferences holds one section: the language and theme selects. Both are
  already in the top bar (`components/layout/Header.tsx` renders
  `LanguageSwitcher` and `ThemeSwitcher`), so the tab is a duplicate.
- The sidebar entry is called "User" and opens `/settings/user`, which is that
  empty-ish page; the real content is on the Security tab.
- `pages/settings/security.tsx` is 719 lines: password, sessions, 2FA, passkeys
  in one scroll, and it is the biggest file of the pages.

## Where

- `resources/ts/pages/settings/user.tsx`, route `userSettings` in
  `app/pages.ts` (`/settings/user`), lazy import in `app/router.tsx`.
- `components/features/spaces/UserSettingsTabs.tsx` (tabs list; the file is in
  `spaces/` but is not about spaces).
- `components/layout/Sidebar.tsx` `settingsItems`: `{ to: '/settings/user',
  labelKey: 'user', also: ['/settings/security', '/settings/api'] }`.
- `userTabLabel` / `UserTab` in `lib/labels.ts`; strings `settings:user.*`,
  `nav:user` in both locales.
- Passkey code in `pages/settings/security.tsx`: state at the top of
  `SecuritySettingsPage`, the `<Section>` around line 400, the add-passkey
  dialog, `useRegisterPasskey` / `useDeletePasskey`.
- Anything that links to `/settings/user` (search `docs/` and
  `scripts/screenshots` too; screenshot pages come from `app/pages.ts`).

## Fix

1. Remove `pages/settings/user.tsx`, the `userSettings` page entry, its lazy
   route and the Preferences tab. Add a redirect `/settings/user` →
   `/settings/security` so old bookmarks work.
2. Rename the sidebar entry and the page title to "Security"
   (`nav.user` → `nav.security`; ru: "Безопасность"). Point it at
   `/settings/security`, `also: ['/settings/api', '/settings/passkeys']`.
3. Split `security.tsx`: move the passkeys section and its dialog into
   `pages/settings/passkeys.tsx` (or a `features/passkeys/` component) and add a
   "Passkeys" tab (`/settings/passkeys`) between Security and API. Register
   the route in `app/pages.ts` with `scope: 'user'`, so the screenshot runner
   picks it up. Hide the tab when `browserSupportsWebAuthn()` is false, as the
   section is hidden now.
4. Move `UserSettingsTabs` out of `features/spaces` into `features/settings`.
5. Update `docs/` wherever "Settings → User" appears.

## Tests (write first)

- Routing: `/settings/user` redirects to `/settings/security`.
- `UserSettingsTabs` renders Security, Passkeys, API (no Preferences).
- Passkey tests currently in the security page's test (if any) move with the
  code and still pass.
- `lib/i18n-keys.test.ts` passes (removed keys gone from both locales).

## Done when

The sidebar says "Security", there is no Preferences tab, passkeys have their
own tab, and `security.tsx` is well under 500 lines.
