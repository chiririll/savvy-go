# 11 — Rework sync groups ("space linking" of categories and tags)

The note calls it "space linking". The code has two things: **links** (signed
cross-space transfers, `Links` in `pages/settings/space.tsx`) and **sync
groups** (shared categories and tags). The complaints (categories, mapping,
absorbing) are about sync groups. The two parts below can ship separately.

## Problem

- One dialog does everything: `SyncReconcileDialog` shows every difference of
  categories, tags, icon/color variants, successors for dropped categories and
  per-type defaults on a single scrolling screen.
- No proper mobile layout: plain `Dialog` (`sm:max-w-2xl`, selects
  `w-full sm:w-72`) instead of `ResponsiveDialog`, which turns into a sheet on
  phones.
- Category names render raw (`{c.name}`), so preset categories show as
  `#RENT` instead of the translated name (the translation lives in
  `locales/*/defaults.json`; other screens translate, this one does not).
- Screenshot tests never reach it: the screenshot capture opens dialogs only
  from elements marked with `testId()` (`scripts/screenshots/capture/dialogs.ts`,
  `lib/test-id.ts`); `SyncGroupSection.tsx` and `SyncReconcileDialog.tsx` have
  none, `pages/settings/space.tsx` has two. And the demo seed creates no sync
  group (`internal/seed/seed.go` links spaces and invites, but does not sync),
  so the page never has a state worth shooting.

## Where

- `resources/ts/components/features/spaces/SyncGroupSection.tsx` (group
  members, per-kind switches, "add space" picker),
  `SyncReconcileDialog.tsx` (236 lines), hooks `useSyncGroup`, `useSyncDiff`,
  `useApplySync`, types `SyncChange`, `SyncDecision`, `SyncDiff` in
  `types/spaces.ts`.
- Backend: `internal/httpserver/api/syncgroups/routes.go`,
  `internal/httpserver/sync_groups_test.go`; docs in `docs/spaces.md`
  ("Shared categories and tags").
- Screenshots: `scripts/screenshots/scenarios/flows/pages/space.ts`,
  `capture/dialogs.ts`.

## Fix — A: UX as a stepper

Turn the dialog into a wizard on `ResponsiveDialog`, one decision per step:

1. **Spaces**: which spaces join the group (list of the user's admin spaces,
   checkboxes) and which kinds (categories, tags).
2. **Keep**: for each joined space, which of its categories/tags to keep
   (everything checked by default; a count and "select all/none" per space).
3. **Map / absorb**: for what is left out or differs, choose the category it is
   merged into (successor) — shown only when transactions would be orphaned —
   and the winning icon/color for variants.
4. **Defaults and summary**: pick the default per type, then a summary of what
   will change in each space and the Apply button.
   Nothing is sent before Apply (as now); a step has Back and keeps its state.

Reuse the existing `SyncDiff` payload; the backend contract does not need to
change. If a step needs data the diff does not give (per-space counts), add
fields to the diff response.

## Fix — B: make it testable and visible

1. Use `categoryDisplayName` (whatever the other screens use to translate
   `#TOKEN` names) for every category name in the dialog and section.
2. Add `testId()` marks: the section's "add space" / "turn on" openers and each
   step; extend `capture/dialogs.ts` if steps need a "next" click (or write a
   scenario in `flows/pages/space.ts` that walks the steps and shoots each).
3. Seed a second administered space with differing categories in
   `internal/seed/seed.go` so the dialog has conflicts, plus one existing sync
   group, and expose it through the manifest.
4. Mobile: add a `mobile` viewport shot of each step to the screenshot run.

## Tests (write first)

- Component tests per step (state kept on Back, Apply disabled until a
  required successor is chosen, preset names translated).
- Existing `sync_groups_test.go` keeps passing; add a Go test for any new diff
  field.
- Screenshot scenario passes in desktop and mobile.

## Done when

A user can join spaces to a group step by step on a phone, sees real category
names, and the screenshot run covers every step.
