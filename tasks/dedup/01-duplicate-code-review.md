# 01 — Review duplicated code found after the refactoring

## Problem

The `refactoring` branch is about +57k / −32k lines (against `dev`) with no new
features. A duplicate scan of the tree at `cc52bcc9` found no copy-paste of old
code next to new code, but did find **structural clones**: same shape, different
names. Some are fine (thin wrappers, shadcn primitives), some are boilerplate
that can become one helper. This task is to check each cluster by hand and fix
the ones worth fixing. It is not a request to remove all of them.

## How the list was made

Per file, drop blank lines, braces, imports and comments; replace identifiers,
string literals and numbers with placeholders; hash sliding windows of 12 lines;
report windows that appear in 2+ files. Generated code (`internal/db/sqlc`),
tests and `*.d.ts` are excluded. About 3.2% of 62k meaningful lines land in a
cross-file clone. A plain (non-normalized) 8-line scan finds about 1.0%, almost
all of it in sqlc, so there is no verbatim copy-paste.

"Windows" below is how many 12-line windows two files share, so it is a rough
size of the clone, not an exact line count. The script is
[dups.py](dups.py): run it from the repo root as `python -I tasks/dedup/dups.py 8`
(exact lines) or `NORM=1 python -I tasks/dedup/dups.py 12` (structural). It
needs `origin/dev` to mark new files. The numbers are a lead, not a verdict. **Read the files before changing anything.**

Status of each cluster: **read** = I read the files while writing this and the
description is what the code does; **not read** = only the scan numbers, verify
first.

## Clusters

### A. Backend: per-resource CRUD handlers (read)

`internal/httpserver/api/budgets/budgets.go` ↔ `api/recurring/recurring.go`
(40 windows), `api/automation/automation.go` ↔ `recurring.go` (26) and
`budgets.go` (23).

What repeats, line for line modulo names:

- `index`, `show`, `destroy`: call a repo method, write `500` or `422` with
  `err.Error()`, write `data` or `204`.
- `param`: parse `{id}`, look it up, write `404 "Not found."` when nil. The
  error from `ParseInt` and from `ByID` is discarded (`id, _ :=`, `b, _ :=`), so
  a DB error is reported as 404. Same in both files.
- `decodeX`: decode into `map[string]json.RawMessage`, pre-fill a body struct
  from `base` for PATCH, marshal the raw map back and unmarshal over it, then
  detect which array fields were sent (`HasTagIDs`, `HasCategoryIDs`). This is
  the PATCH merge that `patch.go` is meant to own (the `decodeBudget` comment
  says "see patch.go"); the per-resource copies still do the plumbing.

Questions to answer:

1. Is there a generic helper already (`httpx` or `patch.go`) that `decodeX`
   should call, or does one need to exist? Go generics over `Input` and the
   repo type may cover `index/show/destroy/param`; `decode` differs per resource
   in the field list, which is the part that has to stay.
2. `recurring.go` names its handlers `recurringIndex` etc. inside package
   `recurring`; `budgets.go` uses `index`. Pick one (the short form).
3. The discarded errors in `param` are a behavior question, not just style:
   decide whether a lookup error should be a 500. If so, fix it in the shared
   helper and write the test first.

Check the other `api/*` packages for the same shape before designing the helper
(`accounts`, `categories`, `currencies`, `debts`, `transfer`, `space`).

### B. Backend: `routes.go` boilerplate (read)

`api/automation/routes.go`, `api/reports/routes.go`, `api/transactions/routes.go`
(13 windows each pair). 35 `routes.go` files share this skeleton:

```go
//go:embed openapi.yaml
var spec []byte

func init() {
    ui.AddSpec(spec)
    routing.Register(func(d *routing.Deps) []routing.Mount {
        return []routing.Mount{routing.SpaceWrite(Routes)}
    })
}
```

This is probably the intended registration pattern, and the clone is the cost
of one-package-per-resource. Decide whether a helper such as
`routing.RegisterSpace(spec, Routes)` is worth it, or leave it and note why.
Do not change the `init()` registration order or the `archtest` rules without
checking `internal/archtest` and `internal/httpserver/AGENTS.md`.

### C. Frontend: resource pages (read)

`pages/budgets/index.tsx` ↔ `pages/recurring/index.tsx` (40 windows, 62% of the
recurring file), and `pages/automation/index.tsx` against both (15).

The two read pages differ only in names. Both do: query hook, delete/create/
update hooks, `useReadOnly`, sort, `useResourceFormDialog`, `Page` + `PageHeader`
+ `FeedList` + form dialog. A `ResourcePage` component taking
`{ ns key, useList, compare, groupBy, Row, FormDialog, hooks }` could replace
them, but the `Row` and `FormDialog` prop names differ (`budget=` vs
`recurring=`), so a generic wrapper may cost more clarity than it saves.
Decide by trying it on budgets + recurring and comparing the line count and the
type signature. If it needs `any` or a many-parameter generic, drop it.

### D. Frontend: React Query hooks (read)

`hooks/use-budgets.ts` ↔ `hooks/use-tags.ts` (19 windows). Each file is 45
lines of the same five hooks (`list`, `item`, `create`, `update`, `delete`) with
a different key, API object, form type and toast keys. `use-crud.ts` already has
`useResourceItem` and `useResourceMutation`. Check how many `hooks/use-*.ts`
follow this exact shape (`ls hooks`) and whether a
`createResourceHooks({ key, api, toasts })` factory removes them cleanly.
Watch for hooks that look the same but invalidate more keys.

### E. Frontend: `*FormDialog.tsx` (read two, not read the rest)

`BudgetFormDialog`, `DebtFormDialog`, `RecurringFormDialog`, `TagFormDialog`,
`AccountFormDialog`, `AutomationRuleFormDialog`: 16–24 windows per pair.

`BudgetFormDialog` and `TagFormDialog` are wrappers over `EntityFormDialog`
with `FORM_ID`, a props interface (`entity, open, onOpenChange, onSubmit,
isSubmitting`), title/description from i18n and `toFormValues`. Most of the
clone is the props interface and the title line; the real content
(`toFormValues`, the form component) is different per resource and has to stay.
This is likely **leave as is**, possibly with a shared `EntityFormDialogProps<T,
F>` type. Read `DebtFormDialog`, `RecurringFormDialog`, `AccountFormDialog`,
`AutomationRuleFormDialog` to confirm they have the same shape before deciding.

### F. Frontend: reports tabs (read the top of one)

`pages/reports/tabs/CashFlowTab.tsx` ↔ `OverviewTab.tsx` (21 windows; 10 in the
plain scan, so some lines are identical, not just similar). `CashFlowTab` uses
`useOverviewMetrics`, the same hook Overview presumably uses. Check whether the
three-card metrics row (`MetricCard` + `Skeleton` fallback) is repeated and
move it to one component in `pages/reports/components/`. Also confirm that
`CashFlowTab` needing `useOverviewMetrics` (not a cash-flow-specific hook) is
intended.

### G. Frontend: login, security, onboarding shells (not read)

Pairs from the plain (non-normalized) scan, so the lines are really identical:

- `components/features/login/AuthCardShell.tsx` ↔ `pages/auth/set-password.tsx`
  (7) and ↔ `onboarding/OnboardingLayout.tsx` (6). `git diff -M30%` also marks
  `OnboardingLayout.tsx` as a 50% copy of `AuthCardShell.tsx`: two layouts doing
  the same job.
- `login/TwoFactorStep.tsx` ↔ `security/SecurityLayout.tsx` (6) ↔
  `security/setup2fa/VerifyStep.tsx` (6): three files sharing 6 identical
  windows. Likely a code-entry field or card header that wants to be one
  component.
- `security/SecurityLayout.tsx` ↔ `sso/form-parts.tsx` (6).
- `login/PasswordLoginForm.tsx` ↔ `login/SetPasswordForm.tsx` (4).
- `onboarding/admin-setup/AdminFieldsStep.tsx` ↔ `RestoreStep.tsx` (5).
- `shared/FormActiveField.tsx` ↔ `shared/FormEstimatedField.tsx` (4): two
  switch fields, probably one `FormSwitchField` with a label key.
- `features/transactions/TransactionForm.tsx` ↔
  `form/use-transaction-model.tsx` (7): a new hook file shares 7 windows with
  the old form. Check that the logic moved and was not copied (see "Check for
  leftovers").

For each pair: open both, decide "extract" or "coincidence", and record it.

### H. Backend: smaller leads (not read)

- `internal/store/sqlite/exchange.go` ↔ `lifecycle.go` (4 windows, new files).
- `internal/seed/reference.go` is a 45% copy of `internal/seed/presets.go` in
  rename detection. Confirm `reference.go` is not left over after the split
  (the tags/categories reference data is meant to be one standalone function
  shared by onboarding and the demo seeder; the demo seeder itself stays).

## Not duplicates — leave alone

- `resources/ts/components/ui/*` (`dialog`, `sheet`, `alert-dialog`,
  `dropdown-menu`, `card`, `table`, `empty`, `tabs`, `avatar`): 60–73% shared
  windows, but these are shadcn wrappers over Radix, written to match the
  library. Do not refactor them.
- `internal/db/sqlc/*`: generated.

## Check for leftovers (do first)

Rename detection (`git diff -M30% -C30% --diff-filter=C origin/dev...HEAD`) lists
files that are 30–60% copies of another file. For each, confirm the source file
still has a reason to exist and the moved logic is not live in two places:

- `pages/admin/security.tsx` → `McpAccess.tsx`, `SigningKeys.tsx`
- `pages/onboarding/admin.tsx` → `useAdminSetup.ts`
- `components/shared/DataTable.tsx` → `DataTableParts.tsx`
- `pages/reports/components/StructureChart.tsx` → `structure-chart-options.ts`
- `internal/seed/reference.go` → `presets.go`

## Tests (write first)

Refactors here must be behavior-preserving, so the test is the existing suite
plus one pin per helper:

- Before touching cluster A, check that PATCH keeps omitted fields, clears a
  field sent as `null`, and replaces `category_ids`/`tag_ids` only when present,
  for budgets and recurring. If a test is missing, add it against the current
  code first, then extract the helper.
- If the `param` lookup behavior changes (a repo error becomes 500 instead of
  404), add a failing test for that change on its own.
- Frontend: existing page/dialog tests must pass unchanged. A hook factory
  needs a test that create/update/delete invalidate the same keys and show the
  same toast keys as before.
- `go test ./...` and the frontend suite (see `resources/ts/AGENTS.md`) at the
  end; `internal/archtest` and `lib/code-shape` must stay green.

## Done when

Each cluster A–H has a recorded decision (extracted, left as is with a reason,
or not a duplicate). Extracted helpers have tests, remove at least as many lines
as they add, and do not change behavior. No new `any`, no generic helper that is
harder to read than the code it replaced.

## Out of scope

The automated duplicate check (a test that fails above a clone threshold) is a
separate task; decide thresholds and an allowlist from the result of this one.

## Result

Checked by hand; `go test ./...`, `tsc` and `vitest run` pass.

| Cluster | Decision |
|---|---|
| A | **Extracted.** `httpx.Lookup` (the `{id}` lookup, used by 8 `*Param` funcs) and `httpx.DecodeSent` (PATCH decode over stored values, used by budgets and recurring). Handlers renamed to the short form. A repo error on lookup is now 500, not 404 (unit test in `httpx/lookup_test.go`). Pins added in `patch_test.go` for `category_ids`/`tag_ids` and for unknown ids. `index`/`destroy` stay: dto and the service differ per resource and a helper would only move the lines. The stale "see patch.go" comments are fixed (no such file). |
| B | **Left as is.** Only 18 packages call `ui.AddSpec`, 7 of them with the bare `SpaceWrite(Routes)` form; the rest wire their own `Deps`. A helper would save about 35 lines but make `routing` depend on `ui`. |
| C | **Left as is.** `ResourcePage` would need list hook, three mutation results, compare, groupBy, `renderRow`, `renderDialog`: about 55 lines saved on two pages, about 55 added. Estimated, not built. |
| D | **Left as is.** Only budgets, tags and recurring have the exact five-hook shape; the other files differ in invalidated keys, delete args or extra hooks. A factory would hide `useCreateBudget` from grep. |
| E | **Left as is.** Dialogs already sit on `EntityFormDialog`; what is left is the props interface, whose entity prop name differs. |
| F | **Extracted.** `OverviewMetricCards` (with a test) replaces the card row in `OverviewTab` and `CashFlowTab`. `CashFlowTab` using `useOverviewMetrics` is intended: inflow, outflow and net are fields of that report. |
| G | `AuthCardShell` is now the one card shell: `OnboardingLayout` wraps it (`compact`) and `set-password` uses it. `OtpCodeInput` replaces three copies of the 6-slot field. `SettingsSection` replaces the `Section` copies in `security/` and `sso/`. `FormFooterCheckbox` is shared by `FormActiveField` and `FormEstimatedField`. Coincidence, left: `PasswordLoginForm`/`SetPasswordForm`, `AdminFieldsStep`/`RestoreStep`, `TransactionForm`/`use-transaction-model` (only prop declarations; the logic moved). |
| H | **Not duplicates.** `exchange.go`/`lifecycle.go` share lock-and-lookup lines only. `seed/presets.go` holds data, `seed/reference.go` does the seeding. |
| Leftovers | None: `admin/security.tsx`, `onboarding/admin.tsx`, `DataTable*`, `StructureChart*`, `seed/reference.go` hold no live copy of the moved code. |
