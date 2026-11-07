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

### R1. Overridable dialog footer (refactor)

**Done 7 Nov 2026.** No mockup: the default footer is drawn exactly as before. Decided with the owner: not a single `actions` render prop but three override levels, so the whole footer or either half can be replaced.

- `components/shared/dialogs/FormDialog.tsx`: the footer is built from exported parts that share a context: `FormDialogFooter` (the row, dialog or mobile sheet), `FormDialogFooterStartSlot` (left, filled by `FormDialogFooterStart`, which works as before), `FormDialogActions` (right, default Cancel and Save) made of `FormDialogCancel` and `FormDialogSubmit` (hidden for the read-only role). `useFormDialog()` gives `{ formId, isSubmitting, isEdit, isReadOnly, close }`.
- Props of `FormDialog` and `EntityFormDialog`, each a node or `(ctx) => node`: `footerStart` (replaces the left region), `footerEnd` (the right), `footer` (the whole row; compose it from `FormDialogFooter` and the parts above).
- No portal-style "end" slot for now; add it if a form body has to contribute buttons.
- Tests in `FormDialog.test.tsx` (defaults, each override, reuse of Cancel/Submit, read-only, mobile sheet); stories `CustomFooterEnd`, `CustomFooter`.
- F5 uses `footerEnd` (or `footer`) built from `transaction.actions`; F4e uses `footerEnd` for the restricted form.

### F2. Transaction row and the list

Mockup first.

**Question to answer before starting:** how does a row show several currencies? The mockup draws these options: (a) the main currency big and the rest as a second line; (b) all joined, `−40 EUR · −300 RSD`; (c) the payments' net. Pick one, or let the agent propose.

- Row stays almost as before: icon, description, subtitle (date, category, account), status badge, amount on the right.
- Subtitle category: the category, or "N categories" for several lines (no notes, no lines inside the row).
- Amount: what was spent, from the lines, per currency. Options to draw: (a) the main currency big and the rest as a second line; (b) all joined, `−40 EUR · −300 RSD`; (c) the payments' net. Change shows only in the detail dialog. Transfer: sent and received, as now.
- Account in the subtitle with several payments: first account and "+N".
- Remove the "Part of a payment" badge and `TransactionItemsRow`; update `lib/transaction-description.ts`, `lib/labels.ts`, the list filters (account and category match any payment or line), pending strip cards.
- Row click and its menu are not changed here (F5 decides what a click opens).

**Done 6 Nov 2026.** Decided with the owner: the icon is the budget's (up to two category emojis stacked; now shared as `categoryIcons` in `components/shared`); several currencies show **one value converted to the first currency with an asterisk** (`TransactionAmount`, tooltip with the exact totals; the full list also goes to the F5 dialog; a tooltip does not open by touch); a row with nothing to show (a pending template without a figure) shows a dash. Built: `lib/transaction-money.ts` (`convertAmount`, `rowAmount`, `transferSides`), `lib/transaction-description.ts` (description, `accountLabel` "Cash +N", subtitle with "N categories"), `netTotalsByCurrency` on lines (a transfer's fee is spending), `TransactionRow`, `UpcomingPendingCard`; `TransactionItemsRow`, the payment badge and the group delete are gone; stories and tests. `RecurringRow` adapts its flat shape locally until F4f. The `≈` of an estimated amount is not shown until F4e/F4f read `estimated`. Left red for later tasks: `lib/negative-balance.ts` (F4c) and the files listed under F1 for F4/F6.

Agreed mockup:

```
DESKTOP
(🛒)  Groceries                                   −52.40 EUR
      Today · Cash · Food
(🛒🚌) Market trip              [Pending]          −40.00 EUR*   <- hover: −40.00 EUR / −300.00 RSD
      Yesterday · Cash +1 · 3 categories
(⇄)   Cash → Visa                                 −200.00 EUR
      Mon · Cash                                  +200.00 EUR
(💼)  Template without a figure                          —
MOBILE: the subtitle truncates, the amount stays
```

### F3. Markdown renderer and editor

Mockup first (the toolbar and the preview).

- Shared `Markdown` component: lazy, subset above, tested with lists, task lists, links, `javascript:` URLs, raw HTML and images (dropped).
- Shared `MarkdownField`: textarea with a **toolbar** (bold, italic, list, task list, link, inline code), keyboard shortcuts (`Ctrl+B`, `Ctrl+I`, `Ctrl+K`), Enter continues a list and Enter on an empty item ends it, and a Write/Preview switch. No WYSIWYG.
- The toolbar actions are pure functions `(text, selection) -> (text, selection)` with table tests.
- Stories. No form uses it yet. Independent of F1: can go first.

**Done 7 Nov 2026.** Library: `react-markdown` + `remark-gfm` (decided with the owner). Built in `components/shared/markdown/`: `Markdown` (lazy wrapper, plain-text fallback), `MarkdownContent` (renderer; headings and images dropped, raw HTML never enabled so it shows as escaped text, links `target="_blank" rel="noopener noreferrer"`), `safe-url.ts` (only http, https, mailto, tel and relative links survive), `MarkdownField` (toolbar of icons, Write/Preview switch, `Ctrl+B/I/K`, Enter continues or ends a list). Pure editing functions in `lib/markdown-edit.ts` with table tests. Strings in `forms:markdown.*` (en, ru). Stories for both components; they pass on desktop and mobile (run in a Playwright container on the remote Docker context). Not exported from the `shared` barrel yet: import the files directly so the parser stays out of the entry chunk. Undo (`Ctrl+Z`) after a toolbar action is not guaranteed: the text is set through React state.

Agreed mockup (toolbar is icons only on mobile too, one scrollable row):

```
[B] [I] [• ] [☑] [🔗] [</>]                [ Write | Preview ]
┌──────────────────────────────────────────────────────────┐
│ - milk                                                   │
│ - [ ] eggs                                               │
└──────────────────────────────────────────────────────────┘
Preview: same box, toolbar disabled, rendered text or "Nothing to preview".
```

**Question to answer before starting:** the library. The proposal is `react-markdown` with `remark-gfm`; the alternative is a smaller one that passes the `javascript:` and raw HTML tests (the agent measures both in the bundle). Say if you have a size limit or a preference.

### F4. Transaction form (split into steps)

Each step has its own mockup. The form keeps working after every step: the old form is replaced in F4a, later steps add modes.

**Agreed design (7 Nov 2026).** One layout for every step: it is built once in F4a with single rows, and F4b and F4c only turn on the (+) buttons. The backend needs no change: it already allows change in any currency, refuses a repeated account (`ErrDuplicateAccount`), a zero payment and a net of the wrong sign, and reports `difference` without refusing the save.

```
┌ New transaction ─────────────────────────────┐
│ [Expense] [Income] [Transfer] [To space]     │   Transfer tabs: inactive until F4d
│ [ Title / Description                      ] │   one line, plain text
│ Amount                 Account               │
│ [ 22.70        ] USD   [Cash         ▾]  (X) │   payment: positive amount, currency from the account
│ Balance after: 184.90 USD                    │
│ Amount                 Account               │
│ [ 52.40        ] EUR   [Visa         ▾]  (+) │   (+) on the last row adds a payment (F4c)
│ Balance after: 947.60 EUR                    │
│ Date                   Default category (!)  │   (!) yellow: lines exist and cover less than paid
│ [ 11/11/2011 []] (X)   [ 🛒 Foo      ▾]  (+) │   (X) clears the date = pending; (+) adds a line (F4b)
│ Category          Amount        Currency     │   lines block, from the first (+)
│ [ 🛒 Foo      ▾] [Amount      ] [EUR  ▾] (X) │
│ [ Note, one line, grows       [Pencil]     ] │   compact note; pencil opens the editor
│ [ 🛒 Foo      ▾] [Amount      ] [EUR  ▾] (X) │
│ [ [Bold] [Italic] [List] ...  Edit/Preview ✕] │   open editor; ✕ returns to the compact note
│ [ Note                                     ] │
│ Tags       [ + ]                             │
│ Change                                   (+) │   change is a payment, drawn apart (F4c)
│ Amount                 Account               │
│ [ 52.40        ] EUR   [Cash         ▾]  (X) │
│ Balance after: 947.60 EUR                    │
│                          [Cancel]  [Save]    │
└──────────────────────────────────────────────┘
Mobile: account and amount stay on one row; the account shrinks to icon and truncated name,
the amount keeps its width, (X)/(+) stay at the end. Checked at 360px in the stories.
```

Rules:

- **Form values.** `payments: [{account_id, amount}]` (amounts positive), `change: [{account_id, amount}]`, `lines: [{category_id, amount, currency_id, note}]`, `default_category_id`, plus type, date, description, tag_ids. The simple mode is one payment and no explicit lines. The builder signs the amounts: paid rows negative for an expense and positive for an income, change rows the opposite.
- **Accounts are unique** across all payment and change rows. Each picker hides the accounts used in other rows, (+) is disabled when none is left, and the schema rejects duplicates. No other limit on change: any account, any currency.
- **Change** is only for money that comes back to another account. Blocking errors: a zero amount, and a net of the wrong sign in the base currency (the API refuses both).
- **Line currency** is one of the currencies of the payment and change accounts (`ErrLineCurrency`).
- **Line notes.** A single line has no note field. From two lines on, each line has a compact note (one line, grows) with a pencil that opens the markdown editor (F3 `MarkdownField`); the editor has a cross at the right end of its toolbar that closes it. A line with a saved note opens expanded.
- **First (+) on a category:** the first line takes the whole payment total and the new line starts empty.
- **Default category and the remainder.** With no lines, the default category is the transaction's category. With lines, the field is locked while they cover the payment and unlocks when they cover less; the yellow triangle shows only then, and its tooltip says how much goes to the default category. The builder saves the uncovered part as one more line in the default category (uncategorised if none), so the saved lines cover the payment and `difference` stays empty. Covering is compared in the base currency at the current rates, as the backend does; the remainder line takes the currency of the first payment. If the lines exceed the payment there is no remainder, the triangle becomes a warning about the excess, and saving is never blocked. The remainder is fixed at the rates of the save, like `difference`.
- **Editing** a saved transaction opens all its lines as ordinary lines; the default category is empty and locked. A transaction with one line opens in the simple mode.
- **Tests first** for the pure functions: the payload builder, the remainder, account uniqueness and the net-sign rule.

**F4a. Form shell, schema and the simple case.** Mockup of the whole form, all modes sketched, only the first built. Rewrite `schemas/transactions.ts`, the payload builder (form values to `payments` + `lines`) and the model hooks. Simple mode: one account, one amount, one category, date, description, tags, for expense and income. Remove the items code: `lib/transaction-items.ts`, `TransactionFormItems`, `TransactionItemRow`, `ItemNumberField`, `item-keys` and their tests, and the leg-based `PaymentRows`.

**F4b. Several lines.** "Split by category" turns the one amount into lines: category, amount, currency (when the accounts have several), and from two lines on the markdown note (F3 `MarkdownField`, compact). Sum of lines against the payment shown as a hint.

**F4c. Several payments, change and currencies.** More than one account: signed amounts, one total per currency, change as a payment of the opposite sign. The soft `difference` check as a warning that never blocks saving. Balance preview for each account (`balance-preview.ts`).

**F4d. Transfers.** Transfer between accounts: one amount for one currency; two amounts and the rate for two (`transfer-rate.ts`); an optional fee line; transfer to another space (`SpaceTransferTargetField`, `space-transfer-input.ts`). The form must allow an optional commission on a same-currency transfer (one amount plus a fee line), which F7b uses for "pay the card".

**F4e. Restrictions and pending occurrences.** `editDetails` mode: only the description, tags and line categories and notes are editable, the rest is disabled with the reason from `restrictions`. Confirm of a pending occurrence with real payments and lines (`deferred/`, `EstimatedAmountFields`, `ApplyDeferredDateDialog`).

**F4f. Recurring form.** `RecurringForm` on the same body: template + schedule, one payment with an empty amount, the `estimated` flag, the rule's `error` shown.

**F4g. Form layout rework (income and expense).** Decided with the owner on 7 Nov 2026; it changes the agreed design above for expense and income. Transfers keep the old layout until their own pass.

- Two equal columns for every row: amount | account, date | category, and for lines category | amount. A row's button (remove, clear the date, add a line) sits inside its column and takes room from the field, so the columns never change.
- The `Amount` and `Account` labels are drawn once, and the (+) that adds a payment sits at the end of that label row. Each row has an (X) only. The change section works the same way.
- Under the account, from the start: the balance now, and `current -> new` once an amount is typed (nothing appears or jumps; a pending date shows the balance now only; a template none). The total is a small text at the end of the `Amount` label (always there, no line of its own). A message about the amount stands in the same line as the balance, in the left half. Plain numbers, shortened from ten thousand up (`123.5K`, `2.5M`, `100 тыс.`), no symbol. A balance that goes negative gets a red triangle; no tooltip, the red number says it. Screen readers get the full numbers and the words.
- Amount fields are plain text that takes digits and one decimal separator (a comma works), cut to the decimals of the currency; no spinners. The currency code stays at the end of the field.
- Date and its clear cross together are as wide as the amount; the default category and its (+) as wide as the account.
- One warning for the lines, a line of text under them: they cover too little (the rest goes to the default category) or too much. The triangle on the category label is gone.
- A line note grows with its text up to about seven rows, then scrolls.
- Lines stay `Category | Amount | (X)`.
- Not in this step: transfers; the default category as the default in every category dropdown of the transfer page.

**F4h. No type selector when editing (later).** Asked by the owner on 7 Nov 2026. The Expense / Income / Transfer / To space tabs are shown only when creating. A saved transaction keeps its type: on edit the tabs are gone (the type is still readable from the icon and the title). Check `use-type-change.ts` and the restricted form, and drop what only the edit-time switch used. Mockup: the form without the tab row.

### F5. Transaction detail dialog (view)

Mockup first. A separate view-only dialog, not a read-only form. In `components/shared/dialogs`.

- Header: icon, description, date, status. Amounts per currency. Payments (account, signed amount, change marked). Lines (category, amount, **rendered markdown note**). Tags, rule info, the `difference` warning, the `restrictions` as text.
- Footer through the `actions` override of R1, built from `transaction.actions`: Edit (opens the form; `editDetails` opens it restricted), Confirm, Skip, Duplicate, Delete.
- Works for what the form cannot open: `transfer_review`, a viewer role, skipped.
- `use-transaction-deep-link` follows whatever a click opens.

**Done 7 Nov 2026.** Mockup agreed with the owner. Decisions: a row click opens the view dialog (option a); no amount block in the header (the payments carry the amounts); the rule is a "Recurring" badge linking to the recurring page; a skipped transaction shows only its badge. Built in `components/features/transactions/detail/` (not `shared/dialogs`: it needs the transaction helpers; it uses `ResponsiveDialog` and `Confirm`): `TransactionDetailContent`, `TransactionDetailDialog` (footer from `transaction.actions`, no Duplicate: the row menu has it; Edit only where `formModeOf` allows; Skip and Delete ask first; on a phone Skip and Delete move to a "⋯" menu), `TransactionDetailHost` (mutations, lazy-loaded). `CreateTransactionProvider` gains `openView`; `TransactionRow` and `TransactionList` get `onOpen` (a click), the menu's Edit still opens the form; the dashboard card uses it too. Deep link: `?view=<id>` opens the dialog, `?edit=<id>` still opens the form. `lib/transaction-view.ts` (`isChangePayment`). Stories and strings (`en`, `ru`). Left for later: the dialog does not show a rule's schedule (only the badge).

**Question that was answered:** what does a row click open? (a) The view dialog, with Edit in the row menu and the dialog footer (proposed; it also works for transactions the form cannot open). (b) The form where editing is allowed and the view only for locked ones (as today).

### F6. Other screens

Mockup of each changed screen.

**Search, decided with the owner (7 Nov 2026).** The list search `q` already matches the description, line notes and amounts (`internal/db/filter/search.go`), so the backend needs no change. No ranking for now: the order stays by date. Fields that are not text have their own filters, so `q` does not match category, tag or account names. Marks only in the palette: the matching text of the description in yellow, and under a transaction the part of a line's note around the first hit, also marked (lists stay without notes). A palette result opens the view dialog (`?view=`).

Search built: `lib/search-highlight.ts` (`searchTerms`, `highlightParts`, `flattenNote`, `noteSnippet`, tests), `Highlight` and the snippet in `TransactionSearchPalette`, the palette amount through `rowAmount`/`TransactionAmount`. No story: the palette needs the router and the API.

Agreed mockup:

```
(🛒)  Oat [milk] run                              −12.40 EUR
      Today · Cash · Food
(🛒)  Groceries                                   −52.40 EUR
      Yesterday · Cash · 2 categories
      …oat [milk] 2 l · eggs
```

Screens: import preview (`PreviewParts`, `types/import.ts`), reports that list transactions (top transactions), sync and transfer review (`SyncReconcile*`), dashboard.

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
