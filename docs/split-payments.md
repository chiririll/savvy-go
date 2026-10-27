# Split payments

One purchase is often paid in more than one piece: half in cash and half by card, or in euros with the change handed back in dinars. A transaction has one account and one currency, so such a purchase is saved as a **payment** with several **parts**. Each part is an ordinary transaction, so balances, budgets and reports work on the parts exactly as before.

In the transaction form, pick **Expense**, then choose how it was paid under the amount:

| Paid with | Use it when | What is saved |
|-----------|-------------|---------------|
| One account | the usual case | one expense |
| Several accounts | cash and card, or two currencies | one expense on each account, for the amount that account paid |
| With change | you handed over more than the price and got change back in another currency | an expense of the price, and an exchange of the rest into the change |

## With change

Rent is 450 EUR. You hand over 500 EUR and get 5,900 RSD back.

- **Amount** is the price, 450, from the account you paid with.
- **Handed over** is 500, in the currency of that account.
- **Change received** is 5,900, in the account the change went to.

Two transactions are saved: an expense of 450 EUR, and a transfer of the remaining 50 EUR into 5,900 RSD (a rate of 118). Rent is exactly 450 EUR in every report, whatever today's rate is, and both balances are exact. The form shows the rate you got and warns when it is far from the stored rate, which usually means a typo.

The change has to be in another currency than the one you handed over. If it is in the same currency, enter what you actually paid (the price) as an ordinary expense.

The form covers a price in the currency you handed over. When the price is in the currency of the change (say you hand over euros for a bill in dinars), record it through the [API](api.md) or an assistant over [MCP](mcp.md), which accept any parts.

## In the list

Every part is a row of its own, so filtering by account shows only that account's part. Each part is marked **Part of a payment**. Editing or deleting any of them opens or deletes the whole payment. A part cannot be changed on its own.

## Rules

- A payment has at least two parts, all on the same day, and cannot be dated in the future: it moves balances at once.
- Parts are expenses, incomes (a refund) or transfers between your accounts. Debts and transfers to another space cannot be parts.
- Items (the lines of a receipt) are not available on a split payment.
- All parts are saved together or not at all. Saving a changed payment replaces its parts.
- Automation rules run on each part, as they do on any new transaction.

## API and MCP

`POST /transaction-groups` takes `{"legs": [...]}`, each leg like the body of `POST /transactions`; `GET`, `PUT` and `DELETE /transaction-groups/{uuid}` read, replace and delete a payment. Transactions carry `groupUuid`. They need the same permissions as transactions. The interactive reference at `/api/docs` has the details.

For an assistant, `create_transaction_group`, `update_transaction_group` and `delete_transaction_group` do the same. Their description teaches the pattern above: the price as an expense, the surplus and the change as a transfer with `to_amount`, never a negative amount.
