# 04 Server setup (flow 2)

Depends on 03. Admin only. Creating the first admin sets `onboarding_server_done` to `false` (pending); the SPA (`ServerSetupGate`, inside `AuthProvider`) sends a signed-in admin to `/setup/server` while it is `false`, so a reload, a closed tab or another device make no difference. Unset (null) on servers that never went through onboarding: nothing is shown. It has no draft and no commit: each step calls an endpoint that already exists, and every step is skippable.

## Steps

Two parts, each its own screen (`/setup/server`, `/setup/server/more`, `/setup/server/mail`, `/setup/server/sso`).

1. **Required: public URL.** `PATCH /api/settings` `app_url`, prefilled with the browser address (`window.location.origin`) when unset. Expose `config.Live.Writable()` as `config_writable` in `GET /api/settings`. When it is false, the step shows a note (configured through `config.toml`, see `docs/deployment.md`) instead of the input, and Continue still works.
2. **Optional: a list**, each row a link with a chevron to its own screen and back, marked **Done** when configured (derived from live state, not stored):
   - **Email**: status and test mail (`GET /api/admin/mail`, `POST /api/admin/mail/test`), pointer to `docs/deployment.md#email`. Done when mail is enabled. Not editable here (mail is configured only in `config.toml`).
   - **Single sign-on**: the preset chooser and provider form; after creating one the SPA returns to the list. Done when a provider exists. (The admin pages cannot be linked to: `ServerSetupGate` sends admins back here while setup is pending.)
3. **Finish or skip** sets the server setting `onboarding_server_done` to `true`; the flow is never shown again. Users who are not server admins never see it.

## Tests (first)

- Not offered to non-admins.
- Finishing or skipping sets `onboarding_server_done`; it is not shown again.
- With `config_writable` false the URL step shows the note and no input.
- `config_writable` appears in `GET /api/settings` (and the OpenAPI spec).
