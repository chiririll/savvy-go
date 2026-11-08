# AI assistants (MCP)

Go Savvy has a built-in [MCP](https://modelcontextprotocol.io) server. An AI assistant such as Claude can read your finances and answer "what did I spend on food last month?". By default it is **read-only**; you can also [let it change data](#letting-it-change-data) ("add 12 EUR for lunch today").

Endpoint: `https://your-savvy.example/mcp`

## Connect Claude on the web

Claude's servers call your instance over the internet, so it needs a public `https://` address set as `app_url` (see [Deployment](deployment.md#configuration)). An instance only on your home network is unreachable.

1. As administrator, choose how Claude will identify itself, see [Ways to connect an app](#ways-to-connect-an-app). The simplest is **Allow a new app for 5 minutes** in **Administration → Security**.
2. In Claude, open **Settings → Connectors → Add custom connector** and enter the endpoint. For an [own client](#own-client), open the advanced settings and enter its client ID (and secret).
3. On the Go Savvy page that opens, sign in if needed, **choose the spaces** the app may read and press **Allow access**.

The connection appears in **Account → API tokens** as an API token named `MCP: …`. Revoke it there to cut the assistant off at once.

Other clients: anything that supports remote MCP servers with OAuth connects the same way. Without OAuth, send an API token from **Account → API tokens** as `Authorization: Bearer svy_…`. Tools only work within the token's grants; a missing one answers with an error naming the permission.

## Ways to connect an app

Claude offers three ways to identify itself. The server supports all of them; an administrator decides which are open in **Administration → Security**.

| Way | Setup | Notes |
|-----|-------|-------|
| [Dynamic registration](#who-may-register-an-app) (DCR) | Press **Allow a new app for 5 minutes** | Works with most clients. Each connection creates a registration, so it is open for one app at a time |
| [Metadata document](#metadata-documents) (CIMD) | Add the host to **Apps that identify themselves by URL** | The app's `client_id` is a link to a JSON document it hosts. Nothing is registered on the server by hand |
| [Own client](#own-client) | Create it under **Own client** | You give the app a client ID and, if needed, a secret |

### Metadata documents

A client ID Metadata Document (CIMD) app uses an `https://` link as its `client_id`. The server downloads the document at that link to learn the app's name and redirect addresses.

- **Off until you allow a host.** The list of hosts in **Administration → Security → Apps that identify themselves by URL** starts empty and has no defaults. Only an administrator can add a host. Enter a bare host name such as `claude.ai`, without `https://`, a path, a port or a wildcard; matching is exact. While the list is empty the server does not advertise the feature.
- Removing a host cuts off every app that came from it at once. Its tokens stop refreshing.
- The document must name the same URL as its `client_id`, give 1 to 10 valid redirect addresses, and not ask for a client secret (`token_endpoint_auth_method` must be `none` or missing). It is at most 5 KB.
- The server fetches it safely: `https` only, no redirects, 5 seconds, and never to a private, loopback or link-local address. A host that resolves to one is refused.
- The document is cached for the time its `Cache-Control: max-age` says, kept between 5 minutes and 24 hours (an hour if absent).
- Anyone can write a document that calls itself "Claude". The app's name comes from it, so the consent page also shows the host the app identifies itself from. Check it before you allow.

### Own client

For an app that asks for an OAuth client ID (and optionally a secret) in its own settings. In **Administration → Security → Own client** enter a name and the redirect addresses the app will use, one per line. For Claude on the web that is `https://claude.ai/api/mcp/auth_callback`.

- Tick **Requires a client secret** if the app should prove itself with one. The server then generates a client ID and a secret and shows both **once**; copy them into the app. Without it, only a client ID is made and the app relies on PKCE alone.
- The secret is stored only as a hash. It cannot be shown again; create a new client if it is lost.
- The app sends the secret as `client_secret_basic` (an `Authorization: Basic` header) or `client_secret_post` (a form field) at the token endpoint, for both the first sign-in and every token renewal.
- These clients are not removed when idle. **Disconnect** deletes one like any other app.

## Who may register an app

- An app can register itself only while an administrator has opened the window in **Administration → Security**. It closes after 5 minutes or after one app registers: one app per click.
- While closed, the server does not advertise registration and `/oauth/register` answers `404`. Connected apps keep working.
- The consent page shows where your browser will be sent back to. Check it before you allow.
- Each user then decides for themselves. **Allow access** gives the app that user's own access to the spaces they chose, nothing more. The page offers the same tabs as API tokens: Viewer (default, read-only), Editor, Admin, or Custom. The app registers once; every user can connect it.

**Administration → Security → Registered apps** lists every app, marked as registered itself, metadata document or created by admin, with what the server saw at registration (only apps that registered themselves have these details), so you can check it is the one you expected:

| Detail     | Meaning |
|------------|---------|
| IP address | Where the registration came from. Claude on the web registers from Anthropic's servers, not your browser. Behind a proxy it is only as reliable as the proxy's forwarding headers |
| Client     | The `User-Agent` the app sent |
| Origin     | The `Origin` header, if sent |
| Returns to | The redirect addresses it registered |
| Registered | When |

The server log records the same (`oauth app registered`), every created own client (`oauth client created`), every rejected metadata document (`oauth client metadata rejected`) and every removal (`oauth app removed`). **Disconnect** deletes the app and revokes all its tokens for all users; it must register again in a new window.

## What an assistant can see

It acts as you, but only in the spaces you chose, within your role there and what you ticked. Spaces you create or join later do not exist for it. It works with accounts, categories, tags, currencies, transactions, budgets, debts, automation rules and reports. It cannot see members, settings, links, transfers or recurring transactions.

Access tokens last an hour and renew in the background. A connection unused for 30 days ends.

| Tool | Returns |
|------|---------|
| `list_spaces` | your spaces and your role in each |
| `list_accounts`, `list_categories`, `list_tags`, `list_currencies` | reference data of a space. `list_accounts` takes `type` (one or several, such as `credit`) to narrow the list; a credit card in it comes with its terms, available credit, last statement and status |
| `get_balance`, `get_balance_history` | balances at the end of a day (`at`, default today), or of every day from `from` to `to`, for all active accounts with a total or just `account_ids`: opening balance plus confirmed payments; returned as the balance chart does, with `native_data` in the account's own currency |
| `list_transactions`, `get_transaction` | transactions; filters (dates, type, status, account or accounts, currencies, categories, tags) and free-text search `q` (description, item names, or an amount in the account's currency); pages of up to 100 |
| `get_transactions_summary` | transaction totals |
| `get_transactions_day_totals` | what each day netted per currency, over every transaction of the day; give `dates`, or `start_date` and `end_date`; takes the filters of `list_transactions` |
| `list_reconciliations` | [reconciliation points](api.md) per account and date: `actualBalance`, `expectedBalance`, `difference` (also in the base currency); filter by `account_ids`, `from`, `to` |
| `list_budgets` | budgets with progress |
| `list_debts`, `get_debts_summary` | debts and totals |
| `list_automation_rules`, `get_automation_rule`, `get_automation_rule_logs` | rules and what they did |
| `get_report` | overview, money flow, expenses by category, cash flow, net worth and other [reports](api.md) |

`list_transactions`, `get_transaction`, `list_accounts`, `list_categories`, `list_tags`, `list_budgets`, `list_debts` and `list_currencies` take `fields`: the keys to keep of each item, for example `id`, `date`, `payments.accountId`, `payments.amount` (a dotted path picks inside a nested object or list). It makes the answer small, so a long period fits; without it every payment repeats its account. Misspelled keys of a transaction are refused; elsewhere they match nothing.

Amounts keep their own currency; the assistant is told never to add amounts of different currencies.

## Letting it change data

On the consent page pick **Editor** or **Admin**, or tick Write and Delete under **Custom**. Write adds and edits, Delete removes, each per part of a space (for example write on transactions only). Because this changes real data, **Allow access** asks once more and lists what can change. Nothing is written unless you chose it.

| Tool | Needs | Does |
|------|-------|------|
| `create_transaction`, `create_transactions`, `update_transaction`, `confirm_transaction`, `skip_transaction`, `duplicate_transaction` | transactions: write | record, change or settle a transaction: payments (the signed money that moved, one per account) and lines (what it was for), see [Transactions](api.md#transactions); a line's `note` is markdown. `create_transactions` takes a list of up to 500 and stores all or none in one save, so use it instead of calling `create_transaction` in a loop. Give `external_id` (a receipt number, say) and a retry fails naming the transaction that already has it instead of adding a duplicate; the same id from another source is stored and listed in `externalMatches`, see [External id and duplicates](api.md#external-id-and-duplicates) |
| `find_similar_transactions` | transactions: write | before adding a statement or receipts, finds the transactions already entered that look like each item (same amount in the same currency, a day or so apart, or the same `external_id`) and says why. It stores nothing, but needs write permission because the request has a body |
| `delete_transaction` | transactions: delete | remove a transaction |
| `create_account`, `update_account` | accounts: write | add or change an account |
| `reconcile_account` | accounts: write | pin accounts to their real balance on a date (`date`, `accounts`: list of `account_id`, `actual_balance` as text, optional `note`). It creates no transaction; the answer shows the expected balance and the `difference`, and a nonzero one means transactions are missing before that date. The same account and date is replaced; several accounts are stored all or none |
| `create_credit_card`, `update_credit_card` | accounts: write | a credit card with its limit, statement day and grace days, see [Credit cards](api.md#credit-cards) |
| `create_category`, `update_category`, `set_default_category` | categories: write | add or change a category |
| `create_tag`, `rename_tag` | tags: write | add or rename a tag |
| `create_currency`, `update_currency`, `set_base_currency` | currencies: write | add a currency, change its rate or the base currency |
| `create_budget`, `update_budget` | budgets: write | add or change a budget |
| `create_automation_rule`, `update_automation_rule`, `toggle_automation_rule`, `test_automation_rule` | automation: write | add, change, switch or try a rule |
| `delete_automation_rule` | automation: delete | remove a rule |
| `create_debt`, `update_debt`, `add_debt_payment`, `collect_debt`, `reopen_debt` | debts: write | record a debt or payment, or reopen it |
| `delete_reconciliation` | accounts: delete | remove a reconciliation point |
| `delete_account`, `delete_category`, `delete_tag`, `delete_currency`, `delete_budget`, `delete_debt`, `delete_credit_card` | that part: delete | remove it |

- The tools are the [API](api.md) operations made with the connection's token: same checks, errors and role limits. Amounts are exact decimals sent as text, in the account's currency.
- Delete tools are marked destructive, so clients such as Claude ask before running them.
- Rules run by themselves on later transactions and can move money between accounts. Grant automation write only to an assistant you trust.
- Changes look like yours in the app.

## Taking changes back

An assistant that may add and change data can still make a mistake, and a connection without the Delete permission cannot remove the wrong transaction it just made. So every change an assistant makes is logged, and it can take back **its own** changes without Delete:

| Tool | Needs | Does |
|------|-------|------|
| `list_my_actions` | audit: read | the changes made with this connection, newest first, each saying whether it can be rolled back now (`rollbackable`) and, if not, why (`rollbackBlockedReason`) |
| `rollback_actions` | audit: write | undo entries by `ids`, or a whole bulk call by `batch_id`. A created transaction is removed; a changed or deleted one is put back exactly as it was, with its exact amounts in each currency |

- A connection sees and undoes only the entries it made itself. Another connection's entries, yours from the app, or those of a personal API token look as if they did not exist.
- It is refused, and nothing at all is undone, if the same transaction was changed again after the assistant's change, by anyone: you in the app, another connection, or the assistant itself (it then undoes the newest change first). Whatever you did by hand after it stays.
- Undoing a delete needs the transaction's account, categories and tags to still exist, and its `external_id` not to be used again by another transaction.
- Undoing is itself logged and cannot be undone; make the change again instead.
- `rollback_actions` is marked destructive, so clients ask before running it.
- The Editor tab of the consent page includes the audit permission. Connections made before it existed do not have it until you add it with the pencil, see below.
- For now only transactions are logged (a CSV import as one entry that cannot be rolled back). Not logged yet: accounts, categories and the other parts of a space.

## Changing a connection later

No need to reconnect. In **Account → API tokens**, press the pencil next to the `MCP: …` connection and change spaces or permissions. Removing access is immediate; added access works from the next request, with no new consent screen. Only parts of a space that the tools use can be granted.

The assistant sees only the tools its connection can use (read-only access to transactions shows no writing tools). After a change it may take a moment, or a reconnect, for the assistant to refresh its tool list; meanwhile a call it should no longer make fails with an error naming the missing permission.

## How it works

Sign-in uses OAuth 2.1 with PKCE. Clients identify themselves by dynamic registration, a client ID metadata document or a client ID you created. Discovery documents: `/.well-known/oauth-protected-resource` and `/.well-known/oauth-authorization-server`. Behind a reverse proxy, forward `/mcp`, `/oauth/*` and `/.well-known/*` like everything else.

## Testing the sign-in locally

To check OAuth on a server, including one on `localhost`, run the test app from the source tree:

```bash
go run ./cmd/oauth-probe -server http://localhost:8080
```

1. In **Administration → Security**, allow a new app.
2. Run the command. It registers, prints an address and opens it in your browser. Sign in and press **Allow access**.
3. It catches the redirect on a free `127.0.0.1` port and prints `PASS` or `FAIL` for discovery, registration, code exchange, a spent code being refused, a call to `/mcp` and a token refresh.

- `go tool mage probe:oauth` does the same; set `OAUTH_SERVER` to change the address.
- Pass the address exactly as `app_url` spells it: `localhost` and `127.0.0.1` are different issuers, and the probe stops if they differ.
- Flags: `-no-browser` only prints the address, `-port` fixes the redirect port.
- Disconnect the registered probe under **Registered apps** when done.
