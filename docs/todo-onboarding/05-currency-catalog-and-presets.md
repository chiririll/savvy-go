# 05 Server-wide currency catalog and presets

Two small read endpoints the create-space modal needs before a space exists.

## Currency catalog to server scope

- Add `GET /api/currencies/catalog?base=USD` (any signed-in user), served from `s.rates.Catalog(ctx, base)` with the same item shape (code, name, symbol, decimals, rate in `base`, zero when unknown).
- The space-scoped catalog stays for a space's own currency page (it filters out what the space already has). The modal uses the server one.

## Presets

- They stay in `internal/seed` (`reference.go`). Export them (`seed.DefaultCategories`, `seed.DefaultTags`), keep `seed.Demo` using them, and serve them with `GET /api/presets`. No logic moves into `ledger`.
- Category tokens (`#RENT`, ...) are translated in the SPA like today; `#OTHER` and `#OTHER_INCOME` are the defaults of their type. Tag presets are plain English names, which stays until tags get tokens too.
- Check `store/arch_test.go` allows `httpserver` to import `seed`. If not, serve the presets from a small file in `httpserver` that calls an exported `seed` function.

## Tests (first)

- Server catalog returns the same items as the space catalog without the filtering; works for any signed-in user, rejects anonymous.
- Presets endpoint returns the 19 categories (types, icons, colors, the two defaults) and the tags.
- `seed.Demo` still seeds the same data.
- Both routes are in `httpserver/apidocs/openapi.yaml`.
