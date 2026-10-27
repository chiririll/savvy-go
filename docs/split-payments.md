# Split payments

One purchase is often paid in more than one piece: half in cash and half by card, or in euros with the change handed back in dinars. A transaction has one account and one currency, so such a purchase is saved as a **payment** with several **parts**. Each part is an ordinary transaction, so balances, budgets and reports work on the parts exactly as before.

In the transaction form, pick **Expense**. The **+** next to the account adds a row (amount and account) for another account that paid; only the last row has a **+**, and **×** removes a row. Rows can be in different currencies, and the form shows one total per currency.

## Change

Rent is 450 EUR. You hand over 500 EUR and get 5,900 RSD back.

- **Amount** is what you handed over, 500, from the account you paid with.
- **Add change** (above the items) adds a row for the change: 5,900 and the dinar account it went to. Add one row for each account that got change.

Two transactions are saved: an expense of 500 EUR, and an income of 5,900 RSD on the dinar account. No exchange rate is stored: reports convert each part to the base currency by their own rates.

A payment that has items cannot have more rows or change: the items add up to a single amount.

## In the list

Every part is a row of its own, so filtering by account shows only that account's part. Each part is marked **Part of a payment**. Editing or deleting any of them opens or deletes the whole payment. A part cannot be changed on its own.

## Rules

- A payment has at least two parts, all on the same day, and cannot be dated in the future: it moves balances at once.
- Parts are expenses, incomes (a refund, or change) or transfers between your accounts. Debts and transfers to another space cannot be parts. The form makes expenses and the incomes of the change; a payment with an exchange between accounts, made through the API, cannot be opened in the form.
- Items (the lines of a receipt) are not available on a split payment.
- All parts are saved together or not at all. Saving a changed payment replaces its parts.
- Automation rules run on each part, as they do on any new transaction.

## API and MCP

`POST /transaction-groups` takes `{"legs": [...]}`, each leg like the body of `POST /transactions`; `GET`, `PUT` and `DELETE /transaction-groups/{uuid}` read, replace and delete a payment. Transactions carry `groupUuid`. They need the same permissions as transactions. The interactive reference at `/api/docs` has the details.

For an assistant, `create_transaction_group`, `update_transaction_group` and `delete_transaction_group` do the same. Their description teaches the pattern above: the price as an expense, the surplus and the change as a transfer with `to_amount`, never a negative amount.
