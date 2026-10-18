# Spaces and Roles

All finances live in a **space**: accounts, transactions, categories, tags, currencies, budgets, debts, recurring transactions and automation rules. Spaces are fully separate, each with its own base currency (say, a personal budget, a family one and a side project). Switch spaces at the top of the sidebar.

A new user starts with no space: create one or accept an invitation. Guests cannot create spaces.

## Creating a space

Use **Create a space** in the switcher, or the window shown to a user with no space (it cannot be closed). Steps:

1. **What a space is**: a short explanation.
2. **Name**.
3. **Copy from another space?** (only if you belong to one). Pick a source or the defaults, and tick what to copy: currencies, accounts (without balances), categories, tags. This is a one-time copy; for categories and tags that stay in step, see [shared categories and tags](#shared-categories-and-tags).
4. **Base entities**, all editable: base currency (required) and other currencies from the server's list; accounts with optional opening balance; categories (the default of each type cannot be removed); tags.
5. **Create**: a summary, then the space is made in one step. The last screen has an optional **Invite members** button; people are invited when you press **Continue**. You can also invite later in the space settings.

Progress is kept in your browser only, so you can close the window and return. Nothing reaches the server before the last step.

**Via the API.** `POST /api/spaces` with only a `name` makes an empty space. It can also fill the space in one request: `baseCurrency`, `currencies` (codes), `accounts` (`name`, `type`, `currency`, `initialBalance`), `categories`, `tags`. Rates come from the server's currency data; a currency without a known rate is refused, never stored with a made-up one. Default expense and income categories are always added. The request creates the whole space or nothing.

## Space roles

| Role       | Can |
|------------|-----|
| **Admin**  | everything an editor can, plus members, invitations, linked spaces, settings, backups, renaming and deleting the space |
| **Editor** | change all finance data and run imports |
| **Viewer** | read data and reports only |

A space always keeps one admin: the last admin cannot leave or be demoted while other members remain.

## Server roles

- **Admin**: manages users, single sign-on, signing keys, system settings and server backups. **Administration → Spaces** shows each space's name, size, quota and members, never its finances. Assigning an admin to a space, deleting it or restoring it is written to the space's audit log, visible to its admins.
- **User**: creates spaces, up to the admin's limit.
- **Guest**: registered through an invitation from a non-admin. Joins spaces as editor or viewer, cannot create or administer spaces. A server admin can promote a guest to user.

## Invitations

Space admins invite people from **Settings → Space** or the last screen of space creation. An invitation has a role, is valid for 7 days and works once. Existing users join directly; new users can register through it if the server admin allows.

How it is sent depends on the server:

| Server | Invite by |
|--------|-----------|
| [Email confirmation](email-and-security.md#email-confirmation) required | email only |
| [Email](email-and-security.md#email) set up | email, or a link to copy: you choose |
| No email | a link to copy |

- **By email**: the link goes only to that address, so only that mailbox can use it, and registering confirms the address. To invite someone else, revoke and invite again.
- **By link**: you copy it once and send it yourself. Anyone with it can use it, and the new account starts unconfirmed. Instead of an address you can add an optional note (up to 100 characters, such as "Kim from work") to tell links apart in your list.
- **While creating a space**, people you add are only a list until you finish. Emails go out afterwards and links appear on the last screen. Remove someone before that and nothing is sent.
- An address cannot be invited to a space it is already invited to or belongs to. After a revoked or expired invitation it cannot be invited again for 1–2 hours; a server admin can end the wait in **Administration → System**.

## Linked spaces and transfers

An admin of two spaces can link them in **Settings → Space**; either side's admin can unlink.

Editors of both can then send money between them: in the transaction dialog, choose the spaces tab next to *Transfer*, then the destination space and accounts. Each side uses its own account's currency.

A transfer is stored in both spaces and signed by the server. If a space is restored or the link breaks:

- **Restore from an older backup**: the linked space is untouched. Differing transfers return as **pending** transactions to accept or reject; they do not affect balances.
- **Unlink, import elsewhere or bad signature**: the transfer is **frozen**. It stays in balances but cannot be edited, and syncs again once the spaces are relinked.

## Shared categories and tags

Spaces that should use the same categories and tags can join a **sync group** in **Settings → Space → Shared categories and tags**. It needs no link or transfers and works between any spaces you administer. A space is in one group at most.

Categories and tags are switched on separately. While a kind is on, any add, edit, rename, default change or delete in one space is applied to all spaces of the group in one step: if it fails anywhere, it is done nowhere. Items match by name, ignoring case (categories also by type).

- **Rights.** Changing categories or tags needs the **editor** role in *every* space of the group. Setting up the group, adding a space and switching a kind on need an **admin** of every space involved. Leaving the group needs only an admin of that space.
- **Joining.** Spaces must first become identical. The dialog lists every difference (missing items, different icon, color or default). For each, choose to keep it (adding it where missing) or delete it everywhere, and which space's icon and color win. Nothing changes until you apply.
- **Deleting.** Deleting a category asks for one replacement: transactions in every space move to it (where there are none, it is simply deleted). Deleting a tag removes it from transactions.
- **Defaults.** The default category of each type is shared too.
- **Leaving.** A space that leaves keeps its categories and tags, which stop following the others. A group left with one space ends.
- **Drift.** A crash, a restored backup or an unavailable space can make spaces differ. At every server start groups are reconciled without asking: whatever one space has and another lacks is added, nothing is deleted, and on a disagreement the space with the lowest id wins (icon, color, default category).

## Limits

Server admins set these in **Administration → System**: spaces per user (those they administer), size quota per space (overridable per space), kept backups per space, and whether invitations may register new users.
