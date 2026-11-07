# Searching transactions

## On the Transactions page

- **Search field** at the top of the **Filters** panel: words you type must all match. A word matches a transaction's description or the name of one of its items (any letters, upper or lower case). A number also matches an amount, a transfer's received amount or an item price, in the currency of its account: `1879`, `1 879`, `18.79` and `18,79` all work.
- **Filters** button opens the panel: the search field, date range, accounts, currencies (shown when the space has more than one), categories and tags. The badge counts the active filter groups, and **Clear** removes them. Within one group a transaction needs to match any of the chosen values; different groups all have to match. A transfer matches a currency if either of its accounts uses it.
- The list's **totals** (the Income and Expense figures in the base currency) follow the same search and filters.

## Search palette (desktop)

Press **`/`** anywhere (not while typing in a field or with a dialog open), or click **Search** in the header. Type to see the best matches and their totals.

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

On a phone there is no palette; use the Filters panel, which holds the search field.

## API and MCP

`GET /api/transactions` takes `q`, `account_ids` (`1,2`) and `currency` (`usd,jpy`) next to the existing filters. With `with_summary=1` the totals cover only the matching transactions. The MCP tool `list_transactions` takes the same filters (see [MCP](mcp.md)).

## The transaction form

- **Expense** and **Income** tabs. The **Transfer** tabs are shown but cannot be picked yet; they come back with the next steps of the form.
- One line at the top for the **title or description** (plain text, one line).
- **Amount** and **account**: the currency of the account stands at the end of the amount. Under it the form shows the account's balance now and after saving. A transaction dated in the future, or without a date, is pending and moves no balance, and the form says so instead.
- **Date** (the cross clears it, which makes the transaction pending) and **category**.
- **Split by category.** The **+** beside the category adds lines, so one payment can be for several categories: the first press makes two lines, the first with the whole amount and the second empty, ready to take its part off the first. Each line has a category and an amount; when the accounts use more than one currency, a line also picks its currency. **×** removes a line, and removing all of them returns to a single category.
- **Default category.** Once there are lines, the category field becomes the **default category**. While the lines cover the whole payment it is locked. When they cover less, it opens and a yellow triangle says how much goes to it: that rest is saved as one more line in this category (or without a category if none is set). If the lines add up to more than the payment, the form warns you but still saves.
- **Line notes.** From two lines on, each line has a note: a short field that grows as you type. The pencil opens the markdown editor (bold, italic, lists, task lists, links, code, with a preview), and the cross at the end of its toolbar folds it back. A note that was saved opens in the editor. A transaction with a single line has no line note: the title says what it is. Notes show in the transaction's details, not in the list.
- A transaction with one payment and several lines can be edited in the form. One made through the API or imported with several payments (or a transfer) cannot be edited in the form yet; the app says so and leaves it as it is. A saved transaction whose lines cover less than its payment opens with the lines only and an empty default category; saving it puts the rest in that category (or leaves it without one).
- **Tags**. Saving an expense that takes an account below zero asks for confirmation first.
