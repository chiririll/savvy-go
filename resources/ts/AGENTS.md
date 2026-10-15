# Frontend

React, Vite, react-query, ShadCN/UI, Tailwind 4. Alias `@/` = `resources/ts`.

```bash
npx tsc --noEmit -p tsconfig.ci.json   # types, CI runs it
npm run build                          # -> public/build; release builds embed it (go tool mage build:release)
```

## Layout

- `api/` one file per backend resource. `hooks/use-*.ts` wrap in react-query.
- `components/ui` ShadCN primitives. `shared` reusable. `features/<name>` per domain.
- `pages/<name>` route pages. `schemas/` zod forms. `types/` backend shapes. `stores/` client state.
- `locales/{en,ru}/<namespace>.json` strings.

## Bundle

- Code reachable from entry (`components/layout`, `components/providers`, `app/`) imports concrete file, never `components/shared` or `features/*` barrels. Barrel pulls dnd-kit, zod, react-hook-form into entry chunk.
- Heavy dialogs opened from layout use `lazy()`.

## Onboarding

- `pages/onboarding/`: admin and server setup.
- Create-space modal drafts (`stores/space-draft.ts`) browser-only. No server drafts. Nothing sent before final `POST /api/spaces`.
- "Copy from another space" is one-time copy in SPA.

## i18n

- Colon namespace: `t('forms:key')`, never `t('forms.key')`.
- New string: add to `en` and `ru`. `lib/i18n-keys.test.ts` checks.
- Plurals inline: `{{count, plural(one: …; other: …)}}`. No `key_one`/`key_other` siblings.
- API error toast: `getApiErrorMessage`.

## Backend contract

- Enum-like constants in `types/` mirror backend values. `lib/backend-contract.test.ts` reads Go source, compares. Backend value changes: update here too.
