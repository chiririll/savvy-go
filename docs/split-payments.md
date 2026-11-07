# Split payments

One purchase is often paid in more than one piece: half in cash and half by card, or in euros with the change handed back in dinars. It is still **one transaction**. It has a **payment** for each account that moved money, and one or more **lines** that say what the money was for. Balances follow the payments; budgets and reports follow the lines.

## Paying from several accounts

In the transaction form pick **Expense** (or **Income**). The **+** at the end of the **Amount / Account** labels adds a payment: an amount and another account. **×** removes a row. An account can be used once in a transaction, so each picker leaves out the accounts already taken.

Payments can be in different currencies. The **Amount** label shows a small total for each currency, and under every account the form shows its balance before and after. Saving a transaction that takes an account below zero asks first.

## Change

Rent is 450 EUR. You hand over 500 EUR and get 5,900 RSD back.

- **Amount** is what you handed over, 500, from the account you paid with.
- **Change** (above the tags) takes a row for the money that came back: 5,900 and the dinar account it went to. Add one row for each account that got change.

The transaction has two payments, −500 EUR and +5,900 RSD, and one line of 450 EUR for the category. No exchange rate is stored: reports convert each amount to the base currency by its own day's rates. Change can be in any currency, but it cannot be worth as much as the payments or more, counted in the base currency; the form says so and does not save.

## Several categories

The **+** beside the category adds lines, so one payment can cover several categories (a shop with food and cleaning products). Each line has a category and an amount, and, when the accounts use more than one currency, a currency. From two lines on each line can also have a **note** in markdown, for example a list of what you bought:

```
- milk
- bread, 2 pcs
- [ ] return the bottle
```

A note is shown in the transaction's details, never in the list. Lines that cover less than the payments put the rest into the **default category**; the form warns when the lines cover too little or too much, and still saves.

## In the list

A transaction is one row, however many accounts paid. The amount is what the payments net to; across several currencies the row shows the total converted to the first currency, marked with an asterisk, with the exact amounts in a tooltip. The subtitle says "N categories" when there are several lines. Filtering by account shows every transaction that has a payment on it.

Click the row for the payments, lines and notes, see [Opening a transaction](transactions.md#opening-a-transaction).

## Rules

- Editing a transaction opens all of its payments, change and lines in the same form. Transfers, debt payments and one side of a transfer to another space have their own limits, see [The transaction form](transactions.md#the-transaction-form).
- A payment of zero is refused. A transaction dated in the future, or without a date, is pending and moves no balance.
- The payments and the lines are saved together or not at all.
- Automation rules run once on the transaction.

## Upgrading from v1.0

The first start on v1.1 converts what was saved:

- **Parts of a v1.0 split payment** stay separate transactions, each with one payment. They are not merged; make a new split payment in the form if you want one transaction.
- **Items** of a transaction become a markdown list in the note of its line, one item per line (`- name`). Quantities and prices are dropped.

## API and MCP

A transaction carries `payments` and `lines`, see [Transactions](api.md#transactions). The old `/transaction-groups` endpoints and the `*_transaction_group` tools are gone.
