# 12 — Update the demo data generator for the new features

Do this last: it depends on 05, 08, 10 and 11.

## Problem

The demo seeder (`internal/seed`) builds users, three spaces, accounts,
transactions, budgets, recurring rules, automation, linked spaces with
transfers and open invitations. It knows nothing of what these tasks add, so
screenshots and the demo instance never show it.

## Where

- `internal/seed/seed.go` (users, spaces, `linkAndTransfer`, `invite`),
  `demo.go` (per-space data), `reference.go` (presets), `small.go`,
  `manifest.go` (what the screenshot runner learns).
- Screenshot config: `scripts/screenshots/seed.toml`, `capture/manifest.ts`.
- Tests: `internal/seed/seed_test.go`.

## Fix

1. Spaces get an emoji and color (05) — different for each of the three.
2. Use the new default categories and tags (08) through the existing
   `seedReference`; check `catByName` / `tagByName` lookups in `demo.go`.
3. At least one account per space (10) — already true for demo, assert it.
4. A sync group (11): put Family Budget and Studio (or two new spaces) in one,
   with categories that differ so the reconcile steps have content. Add the
   group to the manifest.
5. Users with an unverified address when mail is on, and the mail section
   state if the screenshot config sets a mailer (02, 03).
6. Passkeys cannot be seeded; skip.
7. Update `docs/development.md` / `docs/scripts.md` if they list what the demo
   contains.

## Tests (write first)

- `seed_test.go`: every space has an emoji, a color and an account; the sync
  group exists with the expected members; manifest lists it.
- Screenshot run (`npm run screenshots`) passes and shoots the new pages.

## Done when

A fresh demo database shows every feature added in this backlog.
