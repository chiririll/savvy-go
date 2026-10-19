# AI assistants (MCP)

Go Savvy has a built-in [MCP](https://modelcontextprotocol.io) server. An AI assistant such as Claude can read your finances and answer "what did I spend on food last month?". By default it is **read-only**; you can also [let it change data](#letting-it-change-data) ("add 12 EUR for lunch today").

Endpoint: `https://your-savvy.example/mcp`

## Connect Claude on the web

Claude's servers call your instance over the internet, so it needs a public `https://` address set as `app_url` (see [Deployment](deployment.md#configuration)). An instance only on your home network is unreachable.

1. As administrator, open **Administration → Security** and press **Allow a new app for 5 minutes**.
2. In Claude, open **Settings → Connectors → Add custom connector** and enter the endpoint.
3. On the Go Savvy page that opens, sign in if needed, **choose the spaces** the app may read and press **Allow access**.

The connection appears in **Settings → Security → API** as an API token named `MCP: …`. Revoke it there to cut the assistant off at once.

Other clients: anything that supports remote MCP servers with OAuth connects the same way. Without OAuth, send an API token from **Settings → Security → API** as `Authorization: Bearer svy_…`. Tools only work within the token's grants; a missing one answers with an error naming the permission.

## Who may register an app

- An app can register itself only while an administrator has opened the window in **Administration → Security**. It closes after 5 minutes or after one app registers: one app per click.
- While closed, the server does not advertise registration and `/oauth/register` answers `404`. Connected apps keep working.
- The consent page shows where your browser will be sent back to. Check it before you allow.
- Each user then decides for themselves. **Allow access** gives the app that user's own access to the spaces they chose, nothing more. The page offers the same tabs as API tokens: Viewer (default, read-only), Editor, Admin, or Custom. The app registers once; every user can connect it.

**Administration → Security → Registered apps** lists every app with what the server saw at registration, so you can check it is the one you expected:

| Detail     | Meaning |
|------------|---------|
| IP address | Where the registration came from. Claude on the web registers from Anthropic's servers, not your browser. Behind a proxy it is only as reliable as the proxy's forwarding headers |
| Client     | The `User-Agent` the app sent |
| Origin     | The `Origin` header, if sent |
| Returns to | The redirect addresses it registered |
| Registered | When |

The server log records the same (`oauth app registered`) and every removal (`oauth app removed`). **Disconnect** deletes the app and revokes all its tokens for all users; it must register again in a new window.

## What an assistant can see

It acts as you, but only in the spaces you chose, within your role there and what you ticked. Spaces you create or join later do not exist for it. It works with accounts, categories, tags, currencies, transactions, budgets, debts, automation rules and reports. It cannot see members, settings, links, transfers or recurring transactions.

Access tokens last an hour and renew in the background. A connection unused for 30 days ends.

| Tool | Returns |
|------|---------|
| `list_spaces` | your spaces and your role in each |
| `list_accounts`, `list_categories`, `list_tags`, `list_currencies` | reference data of a space |
| `list_transactions`, `get_transaction` | transactions; filters (dates, type, status, account, categories, tags); pages of up to 100 |
| `get_transactions_summary` | transaction totals |
| `list_budgets` | budgets with progress |
| `list_debts`, `get_debts_summary` | debts and totals |
| `list_automation_rules`, `get_automation_rule`, `get_automation_rule_logs` | rules and what they did |
| `get_report` | overview, money flow, expenses by category, cash flow, net worth and other [reports](api.md) |

Amounts keep their own currency; the assistant is told never to add amounts of different currencies.

## Letting it change data

On the consent page pick **Editor** or **Admin**, or tick Write and Delete under **Custom**. Write adds and edits, Delete removes, each per part of a space (for example write on transactions only). Because this changes real data, **Allow access** asks once more and lists what can change. Nothing is written unless you chose it.

| Tool | Needs | Does |
|------|-------|------|
| `create_transaction`, `update_transaction`, `confirm_transaction`, `skip_transaction`, `duplicate_transaction` | transactions: write | record, change or settle a transaction |
| `delete_transaction` | transactions: delete | remove a transaction |
| `create_account`, `update_account` | accounts: write | add or change an account |
| `create_category`, `update_category`, `set_default_category` | categories: write | add or change a category |
| `create_tag`, `rename_tag` | tags: write | add or rename a tag |
| `create_currency`, `update_currency`, `set_base_currency` | currencies: write | add a currency, change its rate or the base currency |
| `create_budget`, `update_budget` | budgets: write | add or change a budget |
| `create_automation_rule`, `update_automation_rule`, `toggle_automation_rule`, `test_automation_rule` | automation: write | add, change, switch or try a rule |
| `delete_automation_rule` | automation: delete | remove a rule |
| `create_debt`, `update_debt`, `add_debt_payment`, `collect_debt`, `reopen_debt` | debts: write | record a debt or payment, or reopen it |
| `delete_account`, `delete_category`, `delete_tag`, `delete_currency`, `delete_budget`, `delete_debt` | that part: delete | remove it |

- The tools are the [API](api.md) operations made with the connection's token: same checks, errors and role limits. Amounts are exact decimals sent as text, in the account's currency.
- Delete tools are marked destructive, so clients such as Claude ask before running them.
- Rules run by themselves on later transactions and can move money between accounts. Grant automation write only to an assistant you trust.
- Changes look like yours in the app.

## Changing a connection later

No need to reconnect. In **Settings → Security → API**, press the pencil next to the `MCP: …` connection and change spaces or permissions. Removing access is immediate; added access works from the next request, with no new consent screen. Only parts of a space that the tools use can be granted.

The assistant sees only the tools its connection can use (read-only access to transactions shows no writing tools). After a change it may take a moment, or a reconnect, for the assistant to refresh its tool list; meanwhile a call it should no longer make fails with an error naming the missing permission.

## How it works

Sign-in uses OAuth 2.1 with PKCE and dynamic client registration, as Claude expects. Discovery documents: `/.well-known/oauth-protected-resource` and `/.well-known/oauth-authorization-server`. Behind a reverse proxy, forward `/mcp`, `/oauth/*` and `/.well-known/*` like everything else.

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
