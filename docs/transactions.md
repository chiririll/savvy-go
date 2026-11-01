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
