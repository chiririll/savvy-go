# Spaces and Roles

All finances live in a **space**: accounts, transactions, categories, tags, currencies, budgets, debts, recurring transactions and automation rules. Spaces are fully separate, each with its own base currency: say, a personal budget, a family one and a side project.

Every user starts with a personal space. Switch spaces at the top of the sidebar.

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

Space admins invite people from **Settings → Space** with a link for a chosen role, optionally tied to an email. Existing users join directly; new users can register through the link if the server admin allows it.

## Linked spaces and transfers

An admin of two spaces can link them in **Settings → Space**; either side's admin can unlink.

Editors of both spaces can then send money between them: in the transaction dialog, choose the spaces tab next to *Transfer*, then the destination space and accounts. Each side uses its own account's currency.

A transfer is stored in both spaces and signed by the server. If one space is restored or the link breaks:

- **Restore from an older backup**: the linked space is untouched. Differing transfers come back as **pending** transactions to accept or reject; they do not affect balances.
- **Unlink, import elsewhere or bad signature**: the transfer is **frozen**. It stays in balances but cannot be edited, and syncs again once the spaces are relinked.

## Limits

Set by server admins in **Administration → System**: spaces per user (those they administer), size quota per space (can be overridden per space), kept backups per space, and whether invitations may register new users.
