# HTTP server

- Domain types free of JSON/HTTP. `dto` converts: camelCase keys, optionals present as `null`, money as JSON number at stored scale.
- New route: update `apidocs/openapi.yaml`. Auth routes deliberately not in it.
- Create-space needs `spaces` and `ledger` together, so lives in `spaces_fill.go`. All or nothing.

## Mailer and bootstrap

- Mailer: config.toml only, never DB. Read via `s.mailer()` / `s.mailSettings()`, never `cfg.Mail`. Bad section disables mail, does not stop startup.
- Emailed links are random tokens stored hashed, never signed. Invitation with address is email-only: route never returns its token, so holding it proves the mailbox and registering confirms the address. Failed send deletes the row.
- Revoke expires invitation, keeps row: unaccepted row per address is the cooldown. Hourly `invitations:prune` and admin clear delete it. Never hard-delete on revoke.
- Email-verification gate (`requireVerifiedEmail`) covers API tokens too. Routes an unverified user may still call: `verificationExempt`.
- Bootstrap: first-run restore runs inside `Guard.Register`, so failed restore keeps password. Any other route creating first user must call `Invalidate`.

## API tokens

- Least privilege, fail closed. Grants never exceed owner's role: checked per request, never at creation.
- Delete is separate level, so token can add without removing.
- Space outside token's list is 404, like non-member.
- Unmapped path segment closed to tokens (`spaceSegments`). Space administration, backups, credentials, sync groups are `sessionOnly`.
- Category/tag change in sync group needs token reaching every space of group. `refsync` still checks owner's role in each.
- SPA keeps no resource or preset list. Reads `GET /api/auth/api-token-capabilities`.

## OAuth (MCP clients)

- No per-app approval. Gates: admin's registration window plus each user's consent. Closed: `/oauth/register` is 404, metadata omits `registration_endpoint`, nothing hints it exists.
- Access token is `svy_` API token (`kind=mcp`). Refresh copies grant and spaces from current token, so later edits survive rotation. Revoking token ends connection.
- `/api/oauth/authorize` session-only: token must never mint tokens.

## MCP server (`/mcp`)

- Tools have no data access. Each calls app's own route in-process (`mcpCall`) with caller's bearer token, so roles, spaces, permissions, DTO/money shape are REST API's. Inner call needs fresh chi route context.
- Tool's route must fall under `auth.MCPResources`. New read tool: add to `mcpReadNeeds`. Write tool: one arg struct plus `writeTool` line.
- Money args are strings. Delete tools carry `DestructiveHint`. Write tools leave `ReadOnlyHint` off.
- `tools/list` filtered by middleware, not `RemoveTools`, so stale client's direct call still gets REST error naming missing permission.
- Automation `conditions`/`actions` shape taught in tool schema text. Keep in step with `internal/automation`.
