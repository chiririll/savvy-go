# 05 — Emoji and color for a space

## Problem

A space is only a name. With several spaces the switcher and the settings page
are lists of similar text. Categories already have an emoji and a color; spaces
should too.

## Where

- Schema: table `spaces` in `internal/migrate/sql/server/0001_initial.sql`
  (`id, uuid, name, quota_bytes, created_by, created_at, updated_at`). The
  project is pre-release: edit `0001` in place, no compatibility step for old
  Go databases (memory: pre-release schema).
- Queries and sqlc output: `internal/db` (`InsertSpaceParams`, `UpdateSpace…`);
  regenerate with the project's mage target.
- Repo: `internal/db/repo/spaces/spaces.go` `type Space struct { ID int64;
  UUID, Name string }`, `Insert(ctx, uuid, name, createdBy, now)`.
- Service: `internal/spaces/space.go`; creation with content in
  `internal/spaces/spacefill`.
- API: `internal/httpserver/api/spaces`, DTO in `internal/httpserver/httpx/dto`.
- Linked/synced spaces and signed backups carry the space name; check
  `internal/backup` and `internal/transfers` for any place the name is copied
  or signed, and decide whether emoji/color travel with it (they should not be
  part of what is signed).
- Frontend: `types/spaces.ts`, `api/spaces.ts`, `CreateSpaceModal.tsx` (name
  step), `pages/settings/space.tsx` (rename section),
  `SpaceSwitcher.tsx`, and the category emoji/color picker to reuse
  (`components/features/categories/CategoryFormDialog.tsx`).

## Fix

1. Columns `emoji TEXT` and `color TEXT` (both nullable) on `spaces`;
   `CHECK` color against `#rrggbb` the same way categories do.
2. Carry them through `Space`, `UserSpace`, the create request
   (`spacefill.Fill` or the create body), update endpoint and the DTO. Validate
   emoji as one grapheme (reject long strings), color as `#rrggbb`.
3. Add the picker to the name step of the create modal and to the rename
   section of the space settings; show a colored emoji chip in `SpaceSwitcher`,
   the settings header and the admin spaces table. Fall back to the first
   letter of the name on the color when no emoji is set.
4. Docs: `docs/spaces.md`, API docs `docs/api.md`.

## Tests (write first)

- Repo/service: insert and update round-trip; invalid color and a long emoji
  string are rejected with a 422 on the right field.
- Backup export/import keeps emoji and color; a transfer or sync with another
  space is unaffected.
- Frontend: switcher renders the chip; create modal sends both values.
- `lib/backend-contract.test.ts` still passes if any enum-like value is added.

## Done when

A user picks an emoji and color when creating or renaming a space and sees it
everywhere the space is named.
