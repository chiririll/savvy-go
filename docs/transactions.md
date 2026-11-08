# Searching transactions

## On the Transactions page

- **Search field** at the top of the **Filters** panel: words you type must all match. A word matches a transaction's description or the note of one of its lines (any letters, upper or lower case). A number also matches the amount of a payment or of a line, in the currency of its account: `1879`, `1 879`, `18.79` and `18,79` all work.
- **Filters** button opens the panel: the search field, date range, accounts, currencies (shown when the space has more than one), categories and tags. The badge counts the active filter groups, and **Clear** removes them. Within one group a transaction needs to match any of the chosen values; different groups all have to match. A transfer matches a currency if either of its accounts uses it.
- The list's **totals** (the Income and Expense figures in the base currency) follow the same search and filters.

## Search palette (desktop)

Press **`/`** anywhere (not while typing in a field or with a dialog open), or click **Search** in the header. Type to see the matches.

| Key | Does |
| --- | --- |
| `Tab` | complete the keyword or value being typed |
| `↑` `↓` | choose a suggestion or result |
| `Enter` | open the chosen transaction, or accept the chosen suggestion |
| `Ctrl+Enter` (`Cmd+Enter`) or the *Show all* row | show everything found on the Transactions page |
| `Esc` | close |

Keywords narrow the search. Names are matched without regard to case:

| Keyword | Example |
| --- | --- |
| `account` | `account:Card` |
| `currency` | `currency:usd,jpy` |
| `category` | `category:Food` |
| `tags` | `tags:food,coffee` |
| `type` | `type:expense` (`income`, `expense` or `transfer`) |

Separate several values with commas (`tags:food,coffee`), quote a name with spaces (`account:"Cash USD"`), and repeat a keyword to add values. Values of one keyword are alternatives; different keywords all have to match. A keyword that is unknown, or a name that does not exist, is searched as plain text. The filters the palette applies show in the Filters panel and in the address, so the page can be bookmarked.

Plain words match the transaction's description and the notes of its lines (any case), and a number matches the amount of a payment or line. In the palette, the matching text of the description is marked in yellow. When a word is found in a line's note, the part of the note around it is shown under the transaction, with the match marked. Results are ordered by date, not by where the match is. The palette lists up to eight results; if five or fewer more exist, they are listed too, otherwise the last row says there are more than 13. The palette does not count every match or total them, so it stays fast on a big space.

On a phone there is no palette; use the Filters panel, which holds the search field.

## API and MCP

`GET /api/transactions` takes `q`, `account_ids` (`1,2`) and `currency` (`usd,jpy`) next to the existing filters. With `with_summary=1` the totals cover only the matching transactions. With `count=0` the server does not count every match: `meta.has_more` says whether another page follows and `meta.total` is only what the page holds. The MCP tool `list_transactions` takes the same filters (see [MCP](mcp.md)).

Each transaction in a list carries `rowAmount`, the one figure a row shows, ready to draw: `main` (an amount and its currency), the exact `totals` per currency and `converted`. With several currencies, `main` is in the first of them and adds the others at the current rates, so it is an estimate; the exact amounts are in `totals`. For a transfer `rowAmount` is what left and `receivedAmount` what arrived. It is for display only: nothing that books or checks money reads it.

`GET /api/transactions-day-totals` is what each day netted: income less expense per currency, largest first by value in the base currency. A transfer counts only its fee lines, skipped transactions not at all. It covers every transaction of the day, however many pages the list has. It takes the filters of the list and needs the days: `dates` (`2026-02-01,2026-02-02`), or both `start_date` and `end_date`. The list shows these under each day heading. The MCP tool `get_transactions_day_totals` takes the same.

## Opening a transaction

Click a row to see it whole: the title, date and status, every **payment** (account and amount, change included), every **line** (category, amount and its note, shown as formatted markdown), tags, and a **Recurring** badge that links to the rule. The buttons at the bottom come from what you may do with it: **Edit**, **Confirm**, **Skip** and **Delete** (the last two ask first). **Duplicate** is in the row's menu. On a phone, Skip and Delete are under the **⋯** button. The address `?view=<id>` opens the dialog on a transaction, `?edit=<id>` opens the form.

## The transaction form

- **Expense**, **Income** and **Transfer** tabs, and a square tab for a transfer to a linked space when the space has a link you can write to. Switching to a transfer drops what a transfer cannot have: lines, change and all accounts but the first.
- One line at the top for the **title or description** (plain text, one line).
- **Amount** and **account**: the labels are written once, above the rows. The amount takes digits and one decimal separator (a comma works), no more decimals than the currency has, and the currency of the account stands at the end of it. Under each account the form shows its balance from the start, and once an amount is typed also what it will be after saving (`1.2K → 800`, shortened from ten thousand up), so nothing jumps; a red triangle marks a balance that goes negative. A transaction dated in the future, or without a date, is pending and moves no balance, and the form says so instead.
- **Several accounts.** The **+** at the end of the **Amount / Account** labels adds another account that paid (or received, for an income), and **×** removes a row. An account can be in a transaction only once: each picker leaves out the accounts the other rows use, and the **+** is off when none is left. Rows can be in different currencies; under them the **Amount** label shows a small **total** per currency at its end (always, so nothing appears when you add an account). Saving an expense that takes several accounts below zero asks once, listing every account that would go negative.
- **Change.** An expense has a **Change** section above the tags, for money that came back to you on another account (for example dinars for the euros you handed over). Its **+** (at the end of the labels, once it has a row) adds a row with an account and an amount; the account can have any currency. Change cannot be worth as much as the payments or more, counted in the base currency at the current rates; the form shows that under the section and does not save. An income has no change section unless a saved one has some. A line can be in the currency of a change account too.
- **Date** (the cross clears it, which makes the transaction pending) and **category**.
- **Split by category.** The **+** beside the category adds lines, so one payment can be for several categories: the first press makes two lines, the first with the whole amount and the second empty, ready to take its part off the first. Each line has a category and an amount; when the accounts use more than one currency, a line also picks its currency. **×** removes a line, and removing all of them returns to a single category.
- **Default category.** Once there are lines, the category field becomes the **default category**. While the lines cover the whole payment it is locked. When they cover less, it opens and a warning under the lines says how much goes to it: that rest is saved as one more line in this category (or without a category if none is set). If the lines add up to more than the payment, the same place warns you, and the form still saves.
- **Line notes.** From two lines on, each line has a note: a short field that grows as you type, up to about seven rows. The pencil opens the markdown editor (bold, italic, lists, task lists, links, code, with a preview), and the cross at the end of its toolbar folds it back. A note that was saved opens in the editor. A transaction with a single line has no line note: the title says what it is. Notes show in the transaction's details, not in the list.
- **Editing.** An expense or an income with any number of payments, change and lines can be edited in the form: the payments of the sign of its type are the payments, the others are the change. A transfer between two accounts of the space opens too, when it has at most one fee line in the currency of its source. A saved transaction whose lines cover less than its payments opens with the lines only and an empty default category; saving it puts the rest in that category (or leaves it without one).
- **Only the details.** A transaction the server lets you change only in part opens in a short form instead: a payment on a debt, and one side of a transfer to another space. So does a transaction the form cannot show whole (an unusual mix of accounts), because saving it in the full form would lose part of it. The short form changes the **title**, the **tags** and, for each line, its **category** and **note**. The date, the accounts and the amounts are shown as text, with a note on top that says why they cannot change (for a debt: change the amount in the debt; for a transfer between spaces: change the amount and the date in the transfer). A transfer between spaces that is in review, and the next occurrence of a recurring transaction, do not open at all: the app says why. Confirm or skip the occurrence from the pending list.
- **Transfer.** Two rows of two columns: the **amount sent** and the account the money leaves, then the **amount received** and the **destination account** (the source is left out of its list). Each account shows its balance under it. The swap button, at the end of the labels of the second row, puts each account where the other was, with its amount, and turns the rate over. The **date** and the **transfer rate** share the next row, the rate under the accounts, then the commission and the tags. Type the amount received or the rate and the other follows. In one currency the amount received shows the amount sent and the rate is 1; both fields stay but cannot be changed. A transfer has no category, lines or change. Tags are kept for a transfer between two accounts of this space.
- **Commission.** A transfer between two accounts can have a commission: it is paid from the source on top of the amount sent, in the source's currency, and saved as a line in an expense category (or none), so reports count it as spending. It works in one currency too, for a bank's fee on paying a card. The amount received is not reduced by it.
- **Transfer to another space.** Pick the destination space, then one of its accounts. It is sent through the transfers API with both amounts and a **date**, which is required; it has no commission and no tags, and the destination account's balance is not previewed. It cannot be edited here.
- **Tags**. Saving an expense that takes an account below zero asks for confirmation first.

## Confirming a pending transaction

**Confirm** on a pending transaction asks for the date it is posted on: today, its own date if it is not in the future, or another day (never a future one). If posting takes an account below zero, the app asks first.

For the next occurrence of a recurring transaction the same window also asks for the **real amounts**: its payments (and, in a transfer between currencies, the money that arrives) and its lines come filled with the expected amounts, and you change the ones that differ, for example a bill that came in at another price or a rate that moved. Every payment needs an amount above zero. A single line that covers the payments follows them and is not asked for; with several lines, or payments in several currencies, each line has its own amount. If you leave the amounts as they are, only the date is sent and the expected amounts stand.

## Recurring transactions

A recurring rule makes a pending transaction on a schedule: confirm or skip it from the pending list, and the rule makes the next one. Its form is the transaction form, with these differences:

- **What it makes.** The same fields: type (**Expense**, **Income** or **Transfer**; a rule cannot move money to another space), title, accounts, category or lines, change, commission and tags. There is no date and no balance preview: each occurrence is dated by the day it falls on, and nothing moves until it is confirmed. The **Date** of an occurrence can be changed when you confirm it.
- **Schedule.** Every n days, weeks, months or years, with a start and an optional end. A weekly rule takes a day of the week and a monthly one a day of the month; a month with fewer days uses its last day. **Active** pauses the rule when it is off.
- **Estimated amount.** The **Estimated amount** box marks the amount as a guess: the row shows it as **≈**, and you enter the real amount when you confirm each occurrence.
- **Amount received left open.** In a transfer between two currencies the **amount received** can stay empty (its field says "At each run's rate"). Each occurrence then works it out from the amount sent and the rates of its day, and the rule shows **≈**. The form leaves open only this amount. A rule made through the API with another amount left open (an expense whose amount comes from its lines) is shown in the list, but the form does not open it, because saving it would lose that: change it through the API.
- **A rule that stopped.** When an account it uses is deleted, the rule stops and its row says why instead of its details. Open it, fix the accounts and tick **Active**: saving clears the message. A rule the form cannot show whole (an account of it is gone) is not opened, and the dialog says so.
