# API

Connect scripts and external apps with an API token.

1. Open **Settings → User → API** and create a token: a name, `read` or `read-write` access and an optional expiration. The token is shown once.
2. Send it as a bearer token.

Finances belong to a space, so their endpoints are under `/api/spaces/{id}/…`. List your spaces to find the id:

```bash
curl -H "Authorization: Bearer svy_xxxxxxxx" https://your-savvy.example/api/spaces

curl -H "Authorization: Bearer svy_xxxxxxxx" https://your-savvy.example/api/spaces/1/accounts

curl -X POST https://your-savvy.example/api/spaces/1/tags \
  -H "Authorization: Bearer svy_xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"name": "from-script"}'
```

## Access

A token acts as its owner: in each space it can do what the owner's role there allows, and a `read` token can only make `GET` requests. Tokens cannot manage users, identity providers, passwords, 2FA, passkeys, API tokens, settings, backups, members or invitations. Revoke a token in the same screen to cut an app off at once.

A space the owner is not a member of answers `404`, the same as one that does not exist.

## Updates

`PATCH` changes only the fields it sends: omitted fields keep their values, and an explicit `null` clears an optional field such as a description or an end date.

## Reference

Interactive reference (Swagger UI): `/api/docs`. Raw OpenAPI 3 spec: `/api/openapi.yaml`.
