# 06 — Align the members list on the space settings page

## Problem

In the members list the role dropdown sits right after a variable-width block
(name and email), and the remove button is only rendered for other users. So
the role selects of "you" and the others are not in the same column, and the
rows have no leading element to anchor them.

## Where

- `resources/ts/pages/settings/space.tsx` `Members`: each `<li>` is
  `flex items-center justify-between`; the right group holds the role `Select`
  (`h-8 w-32`) and, when `isAdmin && m.userId !== me?.id`, a trash button.
- `components/shared/UserAvatar.tsx` is the avatar component; the note asks
  for a plain user icon (lucide `UserRound`) instead of the avatar.

## Fix

1. Add a leading `UserRound` icon (`size-4`, muted, in a fixed-size rounded
   container so names start at the same x).
2. Make the right group a fixed width: always render the action cell. For the
   current user render an invisible placeholder (`size-9`, `aria-hidden`) in
   place of the trash button, so every select lines up.
3. Keep `min-w-0` + `truncate` on the text block so a long email does not push
   the select. On narrow screens let the row wrap the select under the name
   rather than overflowing.
4. Apply the same row to the viewer (non-admin) case: the `Badge` replaces the
   select and should be right-aligned in the same column.

## Tests (write first)

- Render with three members (me, other, guest) as admin: every row has the
  icon, one select and one action cell (button or placeholder), checked by
  role and `aria-hidden`.
- Non-admin: badges, no buttons.
- Screenshot check (after 11 makes the settings page covered): compare the
  desktop and mobile shots of Settings → Space.

## Done when

All role dropdowns are in one vertical line and names start at one x.
