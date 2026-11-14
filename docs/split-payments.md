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

### Letting the amount follow the categories

Once there are lines, a **lock** shows at the end of each payment amount. It starts **open**: the amount is the sum of the lines in the currency of that account, plus any change in that currency, and cannot be typed. Change a line and the payment follows; that is the way to enter a receipt category by category and let the total come out of it.

Click it to **close** it: the payment is fixed where it is, and the lines split it, the rest going to the default category. Click again to let it follow the lines again. While it is closed, typing in the payment never changes the lines: if they cover too much or too little the form warns, and the rest goes to the default category.

- A payment follows only the lines in **its own currency**. A line in another currency does not move it.
- If **several payments** are in the same currency their locks stay closed, because it would be unclear which one should take the change of a line.
- If **change is in another currency** than a payment (the rent example above: 500 EUR paid, 5,900 RSD back), that payment's lock stays closed and says why. Turning the change into the payment's currency would need an exchange rate, and none is stored, so type what you handed over yourself. A payment already in the currency of the change is not affected.
- A payment cannot follow when there is no line in its currency. If the lines are removed while it follows, it goes back to the amount it had before. If it is stopped for another reason (a second payment in its currency, change in another currency), it keeps the amount it has.
- The lock is not saved. A transaction saved with lines that add up to its payment opens with the lock open. One saved with lines that cover less (the rest went to the default category) opens with it closed, so opening it changes nothing. The change rows have no lock.

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
