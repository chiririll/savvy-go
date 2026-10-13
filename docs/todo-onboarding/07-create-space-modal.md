# 07 Create-space modal (flow 3)

Depends on 05 and 06. One modal, used by the sidebar "create space" action and by `NoSpace` (replacing its one-click create; for a user with no spaces it cannot be dismissed). A guest still sees the "ask for an invitation" text, and a user may instead accept a pending invitation. The modal replaces `CreateSpaceDialog` (lives in `components/features/spaces/`).

## Screens

1. **What a space is**: a short explanation (own finances, own base currency, members and roles, separate from other spaces).
2. **Name** of the new space.
3. **Copy from another space?** Skipped entirely when the user belongs to no other space (so always for the first space). Otherwise: choose a source space or start from the defaults, and tick what to copy: currencies, accounts (without balances), categories, tags. Copying is done in the SPA: it reads the source through the existing endpoints (viewer rights are enough) and only the ticked kinds are put into the draft. It is a one-time copy, not a link (shared categories stay a separate feature: sync groups). Anything not copied starts from the defaults on the next screen.
4. **Base entities**, all editable. Each kind starts from the copy if it was ticked, otherwise from the defaults (the presets for categories and tags, nothing for accounts, no currencies chosen yet):
   - base currency (required) and other currencies, from the server-wide catalog;
   - accounts: name, currency, optional opening balance;
   - categories: rename, remove, add. `#OTHER` and `#OTHER_INCOME` cannot be removed (the type defaults);
   - tags (optional).
   - A copied account whose currency was not copied (or is not in the list) adds that currency, so the draft is always consistent. Copied currencies carry only their code; the rate comes from the rate source at creation.
5. **Create**: a summary, then the button (one `POST /api/spaces`, task 06). Afterwards the modal offers an optional **Invite members** button, which opens the invitation dialog. Extract that dialog from `pages/settings/space.tsx` into a component. Without mail it gives the copyable link.

## Draft (browser only)

- A persisted `zustand` store like `stores/space.ts`, key `savvy-space-draft` plus the user id, with a `version` (a different version drops the draft).
- Saved on every change, loaded when the modal opens, removed after a successful create. Closing the modal keeps it.
- Reads and writes are wrapped in try/catch: without storage the modal still works, just without resume.
- Nothing is sent to the server before the final request. No server table, no endpoints.

## Tests (first)

- Draft saved, restored and cleared on success; broken or old-version storage is ignored.
- Copy step skipped with no other space; each ticked kind prefills (accounts without balances), unticked kinds use the defaults; a copied account pulls in its currency.
- Defaults `#OTHER` / `#OTHER_INCOME` cannot be removed.
- `NoSpace`: a user gets the modal that cannot be dismissed; a guest gets the message.
- Locale keys exist in `en` and `ru`.
