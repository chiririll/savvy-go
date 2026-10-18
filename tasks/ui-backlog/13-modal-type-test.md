# 13 — Test that stops the wrong modal type being used

A "think about" note, so this task starts with a decision.

## Problem

The app has five dialog flavours and nothing stops someone using the wrong
one, such as a form in an `AlertDialog` or a confirmation in a `Dialog`:

- `components/ui/dialog.tsx` (`Dialog`), `alert-dialog.tsx` (`AlertDialog`),
  `sheet.tsx` (`Sheet`), `popover.tsx`.
- Wrappers: `shared/ResponsiveDialog.tsx` (dialog on desktop, sheet on
  mobile), `shared/FormDialog.tsx`, `shared/EntityFormDialog.tsx`,
  `providers/ConfirmDialog.tsx` and the `Confirm` helper.
- Today 15 files use `AlertDialog` directly (`settings/space.tsx`,
  `settings/security.tsx`, `settings/api.tsx`, `InviteForm.tsx`,
  `SyncGroupSection.tsx`, `pages/admin/*`, …), many for confirmations that
  could use `Confirm`. `SyncReconcileDialog`, a long form, uses plain `Dialog`
  and is not mobile-friendly.
- The screenshot runner finds dialogs by `role="dialog"` / `"alertdialog"`
  (`scripts/screenshots/capture/dialogs.ts`), so a wrong type also changes what
  gets shot.

## Where

The frontend already has source-scanning tests built on the TypeScript AST:
`lib/code-shape/scan.ts` with `hard.test.ts` / `audit.test.ts`, and
`lib/backend-contract.test.ts`. Put the new rule next to them, e.g.
`lib/modal-rules.test.ts`.

## Fix

1. Decide the rules (suggested):
   - `AlertDialog` only for yes/no confirmation with no inputs: allowed only
     in `components/providers/ConfirmDialog.tsx`, `shared/*Confirm*` and a
     short allow-list; everything else uses `Confirm`.
   - A dialog that contains inputs (`Input`, `Select`, `Textarea`, a `<form>`)
     must be `ResponsiveDialog`/`FormDialog`, not raw `Dialog` or `AlertDialog`.
   - Raw `Dialog` / `Sheet` imports from `components/ui` only inside the
     wrappers.
2. Write the test with the TypeScript compiler API (as `scan.ts` does): for
   each `.tsx` outside `components/ui`, record imports of the modal
   primitives and JSX children of the modal content, and compare with the rules.
3. Start with an explicit allow-list of the current violations (with a
   comment each) so the test passes today, then fix the list down as screens
   are migrated (`SyncReconcileDialog` is covered by task 11).
4. Document the rule in `resources/ts/AGENTS.md` (use the `edit-agents-md`
   skill first, as the repo requires).

## Tests

The task is the test. Prove it by adding a temporary `AlertDialog` with an
`Input` in a throwaway file and checking the test fails, then removing it.

## Done when

Adding a form inside an `AlertDialog` (or a raw `Dialog`) fails `npm test`
with a message that names the right component.
