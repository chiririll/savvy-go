# Tasks: frontend for the new transaction schema (v1.1)

Status: planned 6 Nov 2026 on `dev`. The backend (payments, lines, restrictions) is done: see [db-rework-tasks.md](db-rework-tasks.md), [transactions-erd.md](transactions-erd.md) and [payments-split.md](payments-split.md). This file replaces the frontend half of task 6 there. Nothing is committed by the agent: the owner commits and reviews between tasks.

Today the frontend is still on the flat model: `types/transactions.ts` has `amount`, `account`, `toAccount`, `items`, `groupUuid`; the form builds split payments out of legs.

## Decisions

- **Text fields.** `transaction.description` is one line of plain text (title, searched, shown in lists). A line's note is **markdown** (the column is `transaction_lines.note`; the API key is `note`). A transaction has no notes field of its own.
- **One line, no line note.** A transaction with a single line shows no note field for it: the transaction description is enough. The note field appears from two lines on. A list of products is a markdown list in a line's note.
- **Markdown subset.** Bold, italic, lists, task lists, links, inline code, line breaks. No raw HTML, no images, no headings. Lightweight library (`react-markdown` with `remark-gfm` or smaller), loaded with `lazy()` so it stays out of the entry chunk (`resources/ts/AGENTS.md`, Bundle). Links open with `rel="noopener noreferrer"`, and `javascript:` URLs are dropped.
- **Lists show no notes.** A row shows categories and how much was spent, never a line's note. Notes are shown in the detail dialog (F5) and the form.

## Rules for every task

- **Mockup first** (every task except F1, which has no visuals): before any code the agent drafts a UI mockup of the task and agrees it with the owner in the task's own conversation. ASCII or an HTML sketch is enough; mobile width is part of it. The agreed mockup is pasted into the task's entry here when the task is done, so the next task can build on it.
- Tests for the logic (schema, payload builders, balance preview, toolbar actions); stories for new components, as the folder already does.
- Strings in `en` and `ru` (`lib/i18n-keys.test.ts`); enum constants in `types/` mirror the backend (`lib/backend-contract.test.ts`).
- User-visible change: update `docs/`.
- Done when `npx tsc --noEmit -p tsconfig.ci.json` and the frontend tests pass.

## Tasks, in order

### B1. Converter writes markdown (backend, small)

**Done 6 Nov 2026.** The line's text stays `note` (decided). It has no length limit in the ledger or the API (plain `TEXT`); documented in `docs/api.md` and `docs/mcp.md`.

- `internal/migrate/upgrade/v1`: v1.0 item names become a markdown list in the line's note (`- name`, one per line) instead of a run-together text. Quantity and price stay dropped. Test in `convert_test.go`.
- Check the line note's length limit in the ledger and the API, and say in `docs/api.md` and `docs/mcp.md` that a line's note is markdown.
- Does not block the frontend; do it before the release.

### F1. Types, API layer and contract (no mockup)

- `types/transactions.ts`: `Payment`, `TransactionLine`, `restrictions`; `Transaction` has `payments`, `lines`, `amount` (null across several currencies), `difference`, `actions` with `editDetails`. Drop `items`, `groupUuid`, `toAccount`, `isEstimated`, `TransactionGroup`.
- `api/`, `hooks/use-transactions.ts`, `test/fixtures.ts`: new shapes; the group endpoints go.
- Constants: `space` account type, `in_review` transfer status, restriction reasons; `lib/backend-contract.test.ts` still matches `0001_base.sql`.
- Helpers in `lib/` for what every screen needs: net amount per currency, spent per currency (from lines), the primary category.
- **Done 6 Nov 2026** (types, API layer, constants, `lib/transaction-money.ts`: `netByCurrency`, `spentByCurrency`, `mainAmount`, `primaryCategory`, `lineCurrency`). Also fixed drift in `backend-contract.test.ts` (`debts.direction`, recurring frequencies and transfer reviews read from Go, `transaction-groups` route gone). `credit` and `space` are in `ACCOUNT_TYPES` with icons and labels; the card terms and state are F7a. Left red for the later tasks, by design: `TransactionRow`, `TransactionItemsRow`, `lib/transaction-description.ts`, `lib/transaction-totals.ts` (F2); `lib/payment-input.ts`, `lib/split-payment.ts`, the form, `CreateTransactionProvider`, `schemas/transactions.ts` (F4a); `ApplyDeferredDateDialog`, `EstimatedAmountFields` (F4e); `pages/transactions/index.tsx` (F2, F5); `TransactionSearchPalette` (F6). `RecurringTransaction` is still the old shape (F4f).
- Blocks every other task. The app may not build between F1 and F2/F4; do them on one branch.

**Question to answer before starting (decide before B1 and F1, it touches both):** you call the line text "description", but the column and the API key are `note`. Keep `note`, or rename it everywhere (column, Go types, API, MCP, docs, frontend) before release? The base is unreleased, so a rename is still cheap, but it must come before F1 so the types are written once.

### R1. Overridable dialog footer actions (refactor)

Mockup first: footers on desktop and mobile for the form (Cancel, Save), the view dialog (Edit, Confirm, Skip, Duplicate, Delete) and a restricted form. Independent of the rest; needed by F4e and F5, so do it before them.

- `components/shared/dialogs/FormDialog.tsx`: the right side of the footer (`FormDialogActions`, hardcoded Cancel and Save/Create today) becomes an `actions` prop of `FormDialog` and `EntityFormDialog`: a render function given `{ formId, isSubmitting, isEdit, close, isReadOnly }`. Left out, it draws what is there now, so no other dialog changes.
- `FormDialogActions` is exported, so an override can reuse it and add buttons beside it. The footer start slot (`FormDialogFooterStart`, used by `FormFooterCheckbox`) stays as it is.
- The desktop dialog and the mobile sheet read the same prop.
- Tests: default footer, an override, the read-only role hides Save; a story with a custom footer. Check that the existing dialogs look the same.

**Question to answer before starting:** I read "move the footer buttons out of the form's left handler" as making the right-hand Cancel/Save overridable, by default drawn as now. Is that right, or did you mean the left slot (`FormDialogFooterStart`), or moving the row-menu actions (Duplicate, Delete) into the form footer?

### F2. Transaction row and the list

Mockup first.

**Question to answer before starting:** how does a row show several currencies? The mockup draws these options: (a) the main currency big and the rest as a second line; (b) all joined, `−40 EUR · −300 RSD`; (c) the payments' net. Pick one, or let the agent propose.

- Row stays almost as before: icon, description, subtitle (date, category, account), status badge, amount on the right.
- Subtitle category: the category, or "N categories" for several lines (no notes, no lines inside the row).
- Amount: what was spent, from the lines, per currency. Options to draw: (a) the main currency big and the rest as a second line; (b) all joined, `−40 EUR · −300 RSD`; (c) the payments' net. Change shows only in the detail dialog. Transfer: sent and received, as now.
- Account in the subtitle with several payments: first account and "+N".
- Remove the "Part of a payment" badge and `TransactionItemsRow`; update `lib/transaction-description.ts`, `lib/labels.ts`, the list filters (account and category match any payment or line), pending strip cards.
- Row click and its menu are not changed here (F5 decides what a click opens).

### F3. Markdown renderer and editor

Mockup first (the toolbar and the preview).

- Shared `Markdown` component: lazy, subset above, tested with lists, task lists, links, `javascript:` URLs, raw HTML and images (dropped).
- Shared `MarkdownField`: textarea with a **toolbar** (bold, italic, list, task list, link, inline code), keyboard shortcuts (`Ctrl+B`, `Ctrl+I`, `Ctrl+K`), Enter continues a list and Enter on an empty item ends it, and a Write/Preview switch. No WYSIWYG.
- The toolbar actions are pure functions `(text, selection) -> (text, selection)` with table tests.
- Stories. No form uses it yet. Independent of F1: can go first.

**Question to answer before starting:** the library. The proposal is `react-markdown` with `remark-gfm`; the alternative is a smaller one that passes the `javascript:` and raw HTML tests (the agent measures both in the bundle). Say if you have a size limit or a preference.

### F4. Transaction form (split into steps)

Each step has its own mockup. The form keeps working after every step: the old form is replaced in F4a, later steps add modes.

**F4a. Form shell, schema and the simple case.** Mockup of the whole form, all modes sketched, only the first built. Rewrite `schemas/transactions.ts`, the payload builder (form values to `payments` + `lines`) and the model hooks. Simple mode: one account, one amount, one category, date, description, tags, for expense and income. Remove the items code: `lib/transaction-items.ts`, `TransactionFormItems`, `TransactionItemRow`, `ItemNumberField`, `item-keys` and their tests, and the leg-based `PaymentRows`.

**F4b. Several lines.** "Split by category" turns the one amount into lines: category, amount, currency (when the accounts have several), and from two lines on the markdown note (F3 `MarkdownField`, compact). Sum of lines against the payment shown as a hint.

**F4c. Several payments, change and currencies.** More than one account: signed amounts, one total per currency, change as a payment of the opposite sign. The soft `difference` check as a warning that never blocks saving. Balance preview for each account (`balance-preview.ts`).

**F4d. Transfers.** Transfer between accounts: one amount for one currency; two amounts and the rate for two (`transfer-rate.ts`); an optional fee line; transfer to another space (`SpaceTransferTargetField`, `space-transfer-input.ts`). The form must allow an optional commission on a same-currency transfer (one amount plus a fee line), which F7b uses for "pay the card".

**F4e. Restrictions and pending occurrences.** `editDetails` mode: only the description, tags and line categories and notes are editable, the rest is disabled with the reason from `restrictions`. Confirm of a pending occurrence with real payments and lines (`deferred/`, `EstimatedAmountFields`, `ApplyDeferredDateDialog`).

**F4f. Recurring form.** `RecurringForm` on the same body: template + schedule, one payment with an empty amount, the `estimated` flag, the rule's `error` shown.

### F5. Transaction detail dialog (view)

Mockup first. A separate view-only dialog, not a read-only form. In `components/shared/dialogs`.

- Header: icon, description, date, status. Amounts per currency. Payments (account, signed amount, change marked). Lines (category, amount, **rendered markdown note**). Tags, rule info, the `difference` warning, the `restrictions` as text.
- Footer through the `actions` override of R1, built from `transaction.actions`: Edit (opens the form; `editDetails` opens it restricted), Confirm, Skip, Duplicate, Delete.
- Works for what the form cannot open: `transfer_review`, a viewer role, skipped.
- `use-transaction-deep-link` follows whatever a click opens.

**Question to answer before starting:** what does a row click open? (a) The view dialog, with Edit in the row menu and the dialog footer (proposed; it also works for transactions the form cannot open). (b) The form where editing is allowed and the view only for locked ones (as today).

### F6. Other screens

Mockup of each changed screen.

**Question to answer before starting:** should the search palette and the list search (`q`) match a line's note? Today `q` matches item names; the proposal is to match the transaction description and line notes, so "milk" finds a receipt with a list of products. The backend filter (`internal/db/filter`) would need to follow, so say yes or no before the mockup.

Screens: import preview (`PreviewParts`, `types/import.ts`), the search palette and `lib/transaction-search.ts` (matches item names today; decide whether it searches line notes), reports that list transactions (top transactions), sync and transfer review (`SyncReconcile*`), dashboard.

### F7. Credit cards (task 11 of the rework)

Moved here from [db-rework-tasks.md](db-rework-tasks.md) task 11, so the card screens are built on the new types and the new transfer form. The backend is task 10 (`/credit-cards`, `ComputeCardState`). Two steps, a mockup each.

**F7a. Card in accounts.** After F1.
- `types/`: `credit` account type, `CreditCard` terms and state (`status`: `ok`, `warning`, `danger`; reasons `overdue`, `over_limit`, `payment_due`); `backend-contract.test.ts` still matches `0001_base.sql`.
- Icon for the type in account lists, pickers and the account form.
- Card fields in the account form: limit, statement day, grace days. **Only `fixed_day` exists**: the backend refuses `first_expense` and a grace period over 60 days, so the form has no "starts when the balance goes below zero" switch (task 11 asked for one). Turning a bank account into a card (`Convert`) is a form action.
- Account page: available credit, spent in the period, statement balance, due date and amount due.
- Progress bar like debts and budgets (reuse that component): `-balance / limit` capped at 100%, the amount due marked on it, status colour (neutral, yellow, red). Large on the card page, compact in lists.
- Status: a yellow triangle for `warning`, a red one for `danger`, on the account in lists and on its page, reason in a tooltip. No dashboard block, badge counters or notifications.
- Tests for the form and the status display.

**F7b. Pay the card.** After F4d (transfers).
- A "pay the card" action opens the transfer form with the amount due filled in. When both accounts share a currency the form shows one amount and an optional "commission" field (no two amounts and rate) and saves payments plus a fee line; the data model is unchanged.
- Depends on the F4d transfer form; the commission field is the same fee line.

Out of scope for good: interest, fee and minimum payment calculation, automatic pending transactions for overdue cards, notifications.

### F8. Docs and finish

Rewrite `docs/split-payments.md` for the new form, update `docs/transactions.md`, release notes (items become a markdown list in a line's note; quantities and prices are dropped). Check that every string has `en` and `ru`, remove dead code and unused locale keys. Then the full check.
