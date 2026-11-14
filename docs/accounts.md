# Accounts and reconciliation

## Tabs of the Accounts page

**Accounts** shows the active accounts; **Inactive** (at `/accounts/inactive`) shows the ones switched off, which appear nowhere else on the page. **Reconciliations** (at `/accounts/reconciliations`) lists the reconciliation points, see below.

## Reconciling balances

A **reconciliation point** says what an account really held at the end of a day: "on 4 Sept the account had 5 586". Use it after reading a bank statement or counting cash. The balance of the account at the end of that date becomes the actual value, like the initial balance but for a date.

- **How balances follow.** Every anchor (the initial balance, then each point) restarts the running balance: the balance on a day is the latest anchor on or before it plus the transactions after the anchor's date. Days before the first point are still computed from the initial balance, so a balance chart can jump at a point. Transactions dated on the point's day are already inside the actual value. For a [credit card](credit-cards.md) the balance, the used amount and the available amount follow the pinned balance.
- **The difference.** For each point Savvy also computes the **expected** balance: the previous anchor (the previous point, or the initial balance) plus the transactions between the two. The **difference** is actual minus expected. A non-zero difference means transactions are missing or wrong in that interval; with several points the gap narrows down to the interval between two of them. A transaction added later with a date before the point does not change the balance after the point (it is pinned), but it changes the expected balance and so can close the gap. Nothing is created for you: add the missing transactions.
- **Rules.** One point per account per date; saving again replaces it. A date cannot be in the future. Debt and space accounts cannot be reconciled. A balance with more decimals than the currency has is refused, not rounded. Several accounts saved together are stored all or none, and **Recent actions** can undo them as one unit. Deleting a point puts the balance back to what the previous anchor gives.

### Reconciling in the app

**Reconcile** (left of **New account**, not shown to read-only members) on the Accounts page opens a dialog:

1. Pick the **date** (today by default).
2. Each account has a row with its computed balance on that date filled in. Type the real balance; the row shows the live difference to the expected one. **×** removes a row, **+ Add account** brings removed ones back.
3. **Save** stores a point for every row left; **Cancel** closes it.

### The Reconciliations tab

The **Reconciliations** tab lists the days that have points, newest first. Each day is a fold showing its date, how many accounts were checked and the total difference in the base currency; open it to see every account with its difference in the account's currency, the expected and actual balance (`expected → actual`, or just one value when they match; a difference of zero is shown as `=`), and the note. The list is made for a phone as much as for a desktop. A day with points and no transactions shows only its **Reconciled** line in the transaction list.

### In the transaction list

A day with points has a line **Reconciled** with the total difference of the day in the base currency (`Reconciled -49.5`, shortened to `1.5K` or `2M` for big sums; just **Reconciled** when it is zero). The **?** button next to it opens the details: per account the actual balance, the expected one, the difference and the note, with a button to delete the point.

### API and MCP

Points are part of the accounts permission, see [Reconciliation points](api.md#reconciliation-points). The MCP tools are `reconcile_account`, `list_reconciliations` and `delete_reconciliation`, see [MCP](mcp.md).
