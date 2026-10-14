# AI assistants (MCP)

Go Savvy has a built-in [MCP](https://modelcontextprotocol.io) server, so an AI assistant such as Claude can read your finances and answer questions like "what did I spend on food last month?". It is **read-only**: an assistant cannot add, change or delete anything.

The endpoint is `https://your-savvy.example/mcp`.

## Connect Claude on the web

Claude's servers call your instance from the internet, so it must be reachable over public `https://`, and `app_url` must be set to that address (see [Deployment](deployment.md#configuration)). An instance that is only on your home network cannot be reached by Claude on the web.

1. As an administrator, open **Administration → Security** and press **Allow a new app for 5 minutes**.
2. In Claude, open **Settings → Connectors → Add custom connector**, enter `https://your-savvy.example/mcp` and add it.
3. Claude opens a Go Savvy page that names the app and lists what it may do. Sign in if needed, then press **Allow access**.

Disconnect it in **Settings → User → API**: the connection is listed there as an API token named `MCP: …`. Revoking it cuts the assistant off at once.

## Who may register an app

A new app can only register itself while an administrator has opened the window in **Administration → Security**. The window closes by itself after 5 minutes, or as soon as one app has registered, so it is one app per click. While it is closed the server does not advertise registration at all and `/oauth/register` answers `404`. Apps that are already connected are not affected and keep working. The consent page shows the address your browser will be sent back to: check it before you allow.

## Registered apps

Opening the window is the administrator's approval of a new app. After that, each user decides for themselves on the consent page: pressing **Allow access** gives the app that user's own read-only access, nothing more. The app registers once and every user can connect it.

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

Any MCP client that supports remote servers with OAuth connects the same way. A client without OAuth support can send an API token from **Settings → User → API** as `Authorization: Bearer svy_…`. Whatever the token's scope, the tools stay read-only.

## What an assistant can see

It acts as you. A connection is a read-only API token, so in every space it sees what your own role there allows, and spaces you are not a member of do not exist for it. Access tokens last an hour and are renewed in the background; a connection that is not used for 30 days ends.

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

## How it works

The sign-in uses OAuth 2.1 with PKCE and dynamic client registration, which is what Claude expects. The discovery documents are `/.well-known/oauth-protected-resource` and `/.well-known/oauth-authorization-server`. Behind a reverse proxy, forward `/mcp`, `/oauth/*` and `/.well-known/*` to Go Savvy like everything else.
