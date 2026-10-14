# Onboarding

Terminology:
- **user**: a login (person). Global, lives in the server DB.
- **account**: a financial container (wallet, bank card) inside a space, with its own currency.

Three flows, no generic tutorial engine. Each is a plain page or modal flow.

1. **Admin setup** (first run only): bootstrap password, then create the admin user. Or restore a server backup instead.
2. **Server setup** (admin, once): an optional checklist.
3. **Create space** (every new space): a modal that builds a filled space with one request.

There is no profile tutorial: a user has no server-side profile settings (the language lives in the browser, `LanguageSwitcher`). Add one when a real setting exists.

There is no way back into the onboarding. Flow 1 only exists while the server has no users, flow 2 is shown once, and anything skipped in it is done in the admin panel.

Drafts are kept in the user's browser only. Flows 1 and 2 have none.

## Tasks

Each task ships and is tested on its own. Write the tests first.

| # | Task | Depends on |
|---|------|------------|
| 01 | [Bootstrap password and register gate](01-bootstrap-password.md) | - |
| 02 | [Remove the automatic personal space](02-remove-personal-space.md) | - |
| 03 | [Admin setup page](03-admin-setup.md) | 01 |
| 04 | [Server setup](04-server-setup.md) | 03 |
| 05 | [Server-wide currency catalog and presets](05-currency-catalog-and-presets.md) | - |
| 06 | [Create a filled space in one request](06-create-space-api.md) | 02, 05 |
| 07 | [Create-space modal](07-create-space-modal.md) | 05, 06 |
| 08 | [Restore a server backup on first run](08-restore-on-first-run.md) | 01, 03 |
| 09 | [Docs](09-docs.md) | all |
| - | [Not implementing yet: dashboard task list](not-implementing-dashboard-tasks.md) | - |

Tasks 01, 02 and 05 are independent and can run in parallel. 08 is the riskiest and goes last.

## What the code looks like today

Checked against the code; the plan is built on this.

- **First run.** `GET /api/auth/status` returns `needs_registration` (users == 0). `POST /api/auth/register` ([auth_handlers.go](../../internal/httpserver/auth_handlers.go)) is open to anyone while there are no users, creates an admin, calls `spaces.Provision` and logs in. It is check-then-create with no lock. SPA: `/setup` ([setup.tsx](../../resources/ts/pages/auth/setup.tsx)), `login.tsx` redirects to it, `AuthProvider` caches the status.
- **Personal space.** `spaces.Provision` ([space.go](../../internal/spaces/space.go)) is called from `authRegister`, `usersStore` ([users.go](../../internal/httpserver/users.go)) and the SSO `ProvisionSpace` callback ([server.go](../../internal/httpserver/server.go)). Guests get none.
- **Zero spaces.** `SpaceGate` ([SpaceGate.tsx](../../resources/ts/components/providers/SpaceGate.tsx)) already shows `NoSpace`, with a guest variant. For other users it creates a space named after the user in one click.
- **Creating a space.** `POST /api/spaces` takes only `name` and calls `spaces.CreateOwned` (checks the admin's space limit). The UI is `CreateSpaceDialog` (name only). A new space has no currencies; the first `ledger.Currencies.Create` becomes the base automatically. `ledger.Accounts.Create` takes an `AccountInput` (type defaults to `cash`, optional initial balance, currency by id).
- **Currency catalog.** `GET /api/spaces/{id}/currencies/catalog` lists what the *space* lacks. The data comes from `ledger.RateSource.Catalog(ctx, base)` (`currencydata`, one per server, held as `s.rates`). Entries have code, name, symbol, decimals and a rate in `base` (zero when unknown). `Currencies.FindOrCreateByCode` creates a currency from the catalog with its rate.
- **Reference data.** `internal/seed/reference.go` has `defaultCategories` (19 tokens such as `#RENT`, translated in the SPA; `#OTHER` and `#OTHER_INCOME` are the defaults of their type) and `defaultTags` (10 plain English names). Only `seed.Demo` uses them.
- **Server backups.** `backups.RestoreServer` replaces the server DB, the spaces and the signing key and reloads the key, which also drops all sessions. Laravel databases go through `restore.Server`, which needs the `app_key` and saves an ordinary backup. All these endpoints sit behind `requireAdmin`, so none works before the first admin exists.
- **Server settings.** `PATCH /api/settings` sets `app_url` (written to `config.toml`) and key/value settings. `config.Live.Writable()` says whether the config file could be written at start; the API does not expose it yet. Mail is edited with `GET`/`PUT /api/admin/mail` (written to `config.toml`, live at once) and tried with `POST /api/admin/mail/test`. SSO providers have their own admin pages and endpoints.
- **Invitations.** `POST /api/spaces/{id}/invitations`; the dialog lives in `pages/settings/space.tsx`.
- **Not there:** any bootstrap password; a stored user language.
- **Deployment.** The deb/rpm unit is `savvy-go.service`, config `/etc/savvy-go/config.toml`, data `/var/lib/savvy-go`; the app logs to stdout, so `journalctl -u savvy-go`. Docker data is `/data`.

## Code layout

- Frontend: `pages/onboarding/` (admin, server) and `components/features/onboarding/` (layout, `api/onboarding.ts`) for flows 1 and 2; the modal lives in `components/features/spaces/` and replaces `CreateSpaceDialog`. `pages/auth/setup.tsx` is removed and the `setup` route in `app/pages.ts` and `app/router.tsx` points at the new page. `login.tsx` and `AuthProvider` keep redirecting on `needs_registration`.
- Backend: new `internal/bootstrap` (password, file, hint). The restore endpoint lives in `httpserver` and uses `backup` and `restore`. Add `bootstrap` to the package list in `internal/AGENTS.md` and to `businessPackages` in `store/arch_test.go`. The extended create-space code needs `spaces` and `ledger` together, so it lives in `httpserver`; keep it in one function so it can be tested.
- New routes go into `httpserver/apidocs/openapi.yaml`. New strings go into `locales/{en,ru}` (both).
