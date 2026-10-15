# Frontend

React, Vite, react-query, ShadCN/UI, Tailwind 4. Alias `@/` = `resources/ts`.

```bash
npx tsc --noEmit -p tsconfig.ci.json   # types, CI runs it
npm run build                          # -> public/build; release builds embed it (go tool mage build:release)
```

## Layout

- `api/` one file per backend resource. `hooks/use-*.ts` wrap them in react-query.
- `components/ui` ShadCN primitives. `shared` reusable pieces. `features/<name>` per domain.
- `pages/<name>` route pages. `schemas/` zod form schemas. `types/` backend shapes. `stores/` client state.
- `locales/{en,ru}/<namespace>.json` strings.

## Onboarding

- `pages/onboarding/`: admin and server setup.
- Create-space modal drafts (`stores/space-draft.ts`) live in the browser only. No server-side drafts; nothing is sent before the final `POST /api/spaces`.
- "Copy from another space" is a one-time copy done in the SPA.

## i18n

- Colon namespace: `t('forms:key')`, never `t('forms.key')`.
- New string: add to `en` and `ru` both. `lib/i18n-keys.test.ts` checks.
- Plurals inline: `{{count, plural(one: …; other: …)}}`. No `key_one`/`key_other` siblings.
- API error toast: `getApiErrorMessage`.

## Backend contract

Enum-like constants in `types/` mirror backend values. `lib/backend-contract.test.ts` reads Go source and compares. Backend value changes: update here too.
