# Spaces and Roles

All finances live in a **space**: accounts, transactions, categories, tags, currencies, budgets, debts, recurring transactions and automation rules. Spaces are fully separate, each with its own base currency: say, a personal budget, a family one and a side project.

No space is created for you: a new user starts with none, and creates one or accepts an invitation to someone else's. Guests cannot create spaces and see a message asking for an invitation. Switch spaces at the top of the sidebar.

## Creating a space

Use **Create a space** in the space switcher, or the window shown to a user who has no space yet (that one cannot be closed). It walks through:

1. **What a space is**: a short explanation.
2. **Name**.
3. **Copy from another space?** Shown only if you belong to another space. Pick a source space, or start from the defaults, and tick what to copy: currencies, accounts (without balances), categories, tags. It is a one-time copy and the spaces stay independent; for categories and tags that stay in step, see [shared categories and tags](#shared-categories-and-tags). Anything not copied starts from the defaults.
4. **Base entities**, all editable: the base currency (required) and other currencies from the server's currency list; accounts with an optional opening balance (a copied account brings its currency along); categories, which start from the defaults (the default category of each type cannot be removed); and tags.
5. **Create**: a summary, then the space is made in one step. You can invite members right away or later in the space settings.

Your progress is kept in your browser only, so you can close the window and come back to it; nothing reaches the server before the last step. It is dropped once the space is created.

The same can be done through the API. `POST /api/spaces` with only a `name` makes an empty space. It can also fill the space in the same request: `baseCurrency`, `currencies` (codes), `accounts` (`name`, `type`, `currency`, `initialBalance`), `categories` and `tags`. Currencies get their rates from the server's currency data; one without a known rate is refused rather than stored with a made-up rate. The default expense and income categories are always added. The request either creates the whole space or nothing. See the API reference for the fields.

## Space roles

| Role       | Can                                                                                           |
|------------|-----------------------------------------------------------------------------------------------|
| **Admin**  | everything an editor can, plus members, invitations, linked spaces, settings, backups, renaming and deleting the space |
| **Editor** | change all finance data and run imports                                                       |
| **Viewer** | read data and reports only                                                                    |

A space always keeps one admin: the last admin cannot leave or be demoted while other members remain.

## Server roles

- **Admin**: manages users, single sign-on, signing keys, system settings and server backups. In **Administration → Spaces** they see each space's name, size, quota and members, but not its finances. Assigning an admin to a space, deleting or restoring it is written to the space's audit log (visible to its admins).
- **User**: creates spaces, up to the admin's limit.
- **Guest**: registered through an invitation from a non-admin. Can join spaces as editor or viewer, but cannot create or administer spaces. A server admin can promote a guest to user.

## Invitations

Space admins invite people from **Settings → Space** with a link for a chosen role, optionally tied to an email. Existing users join directly; new users can register through the link if the server admin allows it. When the server has [email](deployment.md#email) set up and the invitation has an address, the link is emailed too; it is always shown on screen as well.

## Linked spaces and transfers

An admin of two spaces can link them in **Settings → Space**; either side's admin can unlink.

Editors of both spaces can then send money between them: in the transaction dialog, choose the spaces tab next to *Transfer*, then the destination space and accounts. Each side uses its own account's currency.

A transfer is stored in both spaces and signed by the server. If one space is restored or the link breaks:

- **Restore from an older backup**: the linked space is untouched. Differing transfers come back as **pending** transactions to accept or reject; they do not affect balances.
- **Unlink, import elsewhere or bad signature**: the transfer is **frozen**. It stays in balances but cannot be edited, and syncs again once the spaces are relinked.

## Shared categories and tags

Spaces that should use the same categories and tags can be put in a **sync group** in **Settings → Space → Shared categories and tags**. A group is not a link: it needs no transfers and works between any spaces you administer. A space is in one group at most.

Categories and tags are switched on separately. While a kind is on, whatever you add, edit, rename, set as default or delete in one space of the group is done in all of them at once, in one step: if it fails anywhere, it is done nowhere. Items are matched by name (categories also by type, income or expense), ignoring case.

- **Rights.** To change categories or tags of a space in a group you need the **editor** role in *every* space of the group; otherwise the change is refused. Setting up the group, adding a space, and switching a kind on need an **admin** of every space involved. Taking a space out of the group needs only an admin of that space.
- **Joining.** When a group is set up, a space is added or a kind is switched on, the spaces first have to become identical. The dialog lists everything that differs (a category or tag missing from some spaces, a different icon or color, a different default). For each you choose to keep it (and add it where it is missing) or to delete it everywhere, and which space's icon and color win. Nothing changes until you apply.
- **Deleting.** Delete a category in one space and you pick one replacement; in every space of the group the transactions of the deleted category move to it, and where there are none it is simply deleted. Deleting a tag removes it from the transactions that carry it.
- **Defaults.** The default category of each type is shared too.
- **Leaving.** A space taken out of the group keeps its categories and tags, which no longer follow the others. A group left with a single space ends.
- **Drift.** A crash between two spaces, a restored backup or a space that was unavailable can make the spaces differ again. At every server start the groups are brought back together without asking: whatever a space has and another lacks is added there, nothing is ever deleted, and where they disagree the space with the lowest id wins (its icon and color, its default category).

## Limits

Set by server admins in **Administration → System**: spaces per user (those they administer), size quota per space (can be overridden per space), kept backups per space, and whether invitations may register new users.
