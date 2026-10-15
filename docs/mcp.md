# AI assistants (MCP)

Go Savvy has a built-in [MCP](https://modelcontextprotocol.io) server, so an AI assistant such as Claude can read your finances and answer questions like "what did I spend on food last month?". By default it is **read-only**. You can also let it record and change things ("add 12 EUR for lunch today"); that is a separate choice you make when you connect it, see [Letting it change data](#letting-it-change-data).

The endpoint is `https://your-savvy.example/mcp`.

## Connect Claude on the web

Claude's servers call your instance from the internet, so it must be reachable over public `https://`, and `app_url` must be set to that address (see [Deployment](deployment.md#configuration)). An instance that is only on your home network cannot be reached by Claude on the web.

1. As an administrator, open **Administration → Security** and press **Allow a new app for 5 minutes**.
2. In Claude, open **Settings → Connectors → Add custom connector**, enter `https://your-savvy.example/mcp` and add it.
3. Claude opens a Go Savvy page that names the app and lists what it may do. Sign in if needed, **choose the spaces** it may read, then press **Allow access**.

Disconnect it in **Settings → User → API**: the connection is listed there as an API token named `MCP: …`. Revoking it cuts the assistant off at once.

## Who may register an app

A new app can only register itself while an administrator has opened the window in **Administration → Security**. The window closes by itself after 5 minutes, or as soon as one app has registered, so it is one app per click. While it is closed the server does not advertise registration at all and `/oauth/register` answers `404`. Apps that are already connected are not affected and keep working. The consent page shows the address your browser will be sent back to: check it before you allow.

## Registered apps

Opening the window is the administrator's approval of a new app. After that, each user decides for themselves on the consent page: pressing **Allow access** gives the app that user's own access to the spaces they chose, nothing more. The page offers the same permission tabs as API tokens: Viewer (the default, read-only), Editor, Admin, or Custom to pick a level for each part of a space. The app registers once and every user can connect it.

**Administration → Security → Registered apps** always lists every registered app with what the server saw when it registered, so you can tell whether it is the one you expected:

| Detail | Meaning |
|--------|---------|
| IP address | The address the registration came from. Claude on the web registers from Anthropic's servers, not from your browser. Behind a reverse proxy the address comes from the proxy's forwarding headers, so it is only as reliable as the proxy |
| Client | The `User-Agent` the app sent |
| Origin | The `Origin` header, when the app sent one |
| Returns to | The redirect addresses the app registered |
| Registered | When it happened |

The same details are written to the server log (`oauth app registered`), and so is every removal (`oauth app removed`).

**Disconnect** deletes the app and revokes every token it holds, for all users, at once. It has to register again in a new window.

## Other clients

Any MCP client that supports remote servers with OAuth connects the same way. A client without OAuth support can send an API token from **Settings → User → API** as `Authorization: Bearer svy_…`. A tool can only do what the token grants: give it access to the parts the tools use (see the tables below) and to the spaces it should use. A tool the token has no grant for answers with an error that names the missing permission, so a read-only token can use every reading tool and none of the writing ones.

## What an assistant can see

It acts as you, but only in the spaces you chose on the consent page. A connection is an API token, so in those spaces it can do what your own role there allows, and no more than you ticked. Every other space, including ones you create or join later, does not exist for it. It works with accounts, categories, tags, currencies, transactions, budgets, debts and reports, and cannot see members, settings, links, transfers, recurring transactions or automation. Open **Settings → User → API** to see what a connection holds. Access tokens last an hour and are renewed in the background; a connection that is not used for 30 days ends.

| Tool | Returns |
|------|---------|
| `list_spaces` | your spaces and your role in each |
| `list_accounts`, `list_categories`, `list_tags`, `list_currencies` | the space's reference data |
| `list_transactions`, `get_transaction` | transactions, with filters (dates, type, status, account, categories, tags) and pages of up to 100 |
| `get_transactions_summary` | transaction totals |
| `list_budgets` | budgets with progress |
| `list_debts`, `get_debts_summary` | debts and their totals |
| `get_report` | overview, money flow, expenses by category, cash flow, net worth and the other [reports](api.md) |

Amounts keep their own currency: the assistant is told never to add amounts of different currencies.

## Letting it change data

Pick **Editor** or **Admin**, or tick Write and Delete under **Custom**, on the consent page. Write lets it add and edit, Delete lets it remove, each per part of a space (for example write on transactions only). Because this changes your real data, pressing **Allow access** then asks once more and lists what the assistant will be able to change. The default stays read-only; nothing is written unless you chose it.

| Tool | Needs | Does |
|------|-------|------|
| `create_transaction`, `update_transaction`, `confirm_transaction`, `skip_transaction`, `duplicate_transaction` | transactions: write | record an income, expense or transfer, change one, settle a scheduled one |
| `delete_transaction` | transactions: delete | remove a transaction |
| `create_account`, `update_account` | accounts: write | add or change an account |
| `create_category`, `update_category`, `set_default_category` | categories: write | add or change a category |
| `create_tag`, `rename_tag` | tags: write | add or rename a tag |
| `create_currency`, `update_currency`, `set_base_currency` | currencies: write | add a currency, change its rate or the base currency |
| `create_budget`, `update_budget` | budgets: write | add or change a budget |
| `create_debt`, `update_debt`, `add_debt_payment`, `collect_debt`, `reopen_debt` | debts: write | record a debt, a payment on it, or reopen it |
| `delete_account`, `delete_category`, `delete_tag`, `delete_currency`, `delete_budget`, `delete_debt` | that part: delete | remove it |

The tools are the same operations as the [API](api.md), made with the connection's token: the same checks, the same errors, the same limits by role. Amounts are exact decimals sent as text, in the currency of the account. Delete tools are marked destructive, so clients such as Claude ask before running them. Changes made by an assistant look like yours in the app; to see what a connection may do or to cut it off, open **Settings → User → API**.

## How it works

The sign-in uses OAuth 2.1 with PKCE and dynamic client registration, which is what Claude expects. The discovery documents are `/.well-known/oauth-protected-resource` and `/.well-known/oauth-authorization-server`. Behind a reverse proxy, forward `/mcp`, `/oauth/*` and `/.well-known/*` to Go Savvy like everything else.

## Testing the sign-in on your own machine

To check that OAuth works on a server, including one on `localhost`, run the small test app from the source tree:

```bash
go run ./cmd/oauth-probe -server http://localhost:8080
```

1. In **Administration → Security**, allow a new app (the window is 5 minutes).
2. Run the command. It registers itself, prints an address and opens it in your browser. Sign in and press **Allow access**.
3. It catches the redirect on a free `127.0.0.1` port and prints `PASS` or `FAIL` for each step: discovery, registration, code exchange, a spent code being refused, a call to `/mcp`, and a token refresh.

`go tool mage probe:oauth` does the same; set `OAUTH_SERVER` to change the address.

Pass the address exactly as the server's `app_url` spells it: `localhost` and `127.0.0.1` are different issuers, and the test app stops if they differ. The app it registers shows up under **Registered apps**; disconnect it when you are done. Use `-no-browser` to only print the address, `-port` to fix the redirect port.
