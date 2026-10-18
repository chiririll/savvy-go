# 07 — One invitation list, above the form, on every screen

## Problem

Settings → Space → Invitations shows the same invitations twice:

1. `InviteForm` keeps a session-only list `sent` (every invitation made since
   the page loaded, with Copy and Cancel buttons), drawn above its fields.
2. `Invitations` in `pages/settings/space.tsx` then draws the server's list
   `open` (from `useSpaceInvitations`) **below** the form, with Revoke.

A new invitation is therefore in both lists at once, and the list order is
"session list, form, server list". The Copy button exists only in the session
list: reload the page and the link of a pending invitation can never be copied
again (the server stores only `token_hash`).

## Where

- `resources/ts/components/features/spaces/InviteForm.tsx` (session list,
  cancel dialog, `add`), `InviteFields.tsx` (inputs), `InvitationRow.tsx`
  (shared row).
- `resources/ts/pages/settings/space.tsx` `Invitations` (server list).
- Other screens with invitations:
  - Create-space modal: `create/MembersStep.tsx` (draft list above
    `InviteFields`; this one is already "list, then form").
  - `components/features/spaces/ClearInvitationsSection.tsx` (admin → System).
  - `pages/invite.tsx` (accepting side), `components/features/users/UserPasswordLinkDialog.tsx`
    and the admin user invite flow in `pages/admin/users.tsx`.
  - `api/spaces.ts`, `hooks/use-spaces.ts` (`useInvite`, `useRevokeInvitation`).

## Fix

1. One list, rendered from the server's `useSpaceInvitations` data, **above**
   the form, using `InvitationRow` (email or note, role, expiry; Revoke with the
   existing confirm dialog).
2. Remove `sent` from `InviteForm`. The Copy action cannot come from the
   server list (no token). Instead show the new link once, right after
   creating a link invitation: a highlighted row on top with Copy and a "this
   link is shown only once" note that disappears on reload. Keep that one row
   in `InviteForm` state and drop it when the server list gains the same id.
3. Make `Invitations` the only owner of the list and pass `onCreated` to the
   form, or move both into one `SpaceInvitations` component.
4. Review the four other screens listed above for the same pattern. Create-space
   `MembersStep` already has "list then form": share the row and empty-state
   with it so the three places look alike.
5. Update `docs/spaces.md` if it describes the screen.

## Tests (write first)

- Create an invitation: exactly one row for it in the DOM, after the server
  list refetches; the once-only Copy row shows for link invitations only.
- Reload: the row is in the list, no Copy button.
- Revoke removes the row; cancel dialog text is unchanged.
- `MembersStep.test.tsx` still passes with the shared row.

## Done when

An invitation appears once, in a single list above the form, on all screens.
