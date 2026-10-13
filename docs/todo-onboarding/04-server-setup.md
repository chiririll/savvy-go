# 04 Server setup (flow 2)

Depends on 03. Admin only, shown right after admin creation. It has no draft and no commit: each step calls an endpoint that already exists, and every step is skippable.

## Steps

- **Public URL**: `PATCH /api/settings` `app_url`. Expose `config.Live.Writable()` as `config_writable` in `GET /api/settings`. When it is false, the step does not show the input; it shows a note that the app is configured through `config.toml` (or the file must be made writable), with a pointer to `docs/deployment.md`.
- **Email**: status and test mail (`GET /api/admin/mail`, `POST /api/admin/mail/test`), with a pointer to `docs/deployment.md#email`. Not editable here (mail is configured only in `config.toml`).
- **Single sign-on**: a link to the existing providers page.
- **Finish or skip** sets the server setting `onboarding_server_done` (add it to `settings.IsServerKey`); the flow is never shown again. Users who are not server admins never see it.

## Tests (first)

- Not offered to non-admins.
- Finishing or skipping sets `onboarding_server_done`; it is not shown again.
- With `config_writable` false the URL step shows the note and no input.
- `config_writable` appears in `GET /api/settings` (and the OpenAPI spec).
