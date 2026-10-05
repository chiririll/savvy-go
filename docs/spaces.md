# Spaces and Roles

All finances in Go Savvy live in a **space**: its accounts, transactions, categories, tags, currencies, budgets, debts, recurring transactions and automation rules. Spaces are fully separate from each other, so one can be your personal budget, another the family's and a third a side project, each in its own base currency.

Every user starts with a personal space. Switch between spaces at the top of the sidebar; the page you are on stays open and shows the other space's data.

## Space roles

Each member of a space has one of three roles there:

| Role       | Can                                                                                                    |
|------------|--------------------------------------------------------------------------------------------------------|
| **Admin**  | everything below, plus members and invitations, linked spaces, space settings, backups, renaming and deleting the space |
| **Editor** | add and change transactions, accounts, categories, tags, currencies, budgets, debts, recurring transactions, automation rules and imports |
| **Viewer** | read data and reports; the app hides everything that writes                                            |

A space always keeps at least one admin: the last admin cannot leave or be demoted while other members remain.

## Server roles

The instance itself has its own roles:

- **Admin** manages users, single sign-on, signing keys, system settings and server backups. In **Administration → Spaces** they see every space's name, size, quota and members, but never its finances. They can assign an admin to a space; that action, like deleting or restoring a space, is logged in the space's audit log, which its admins see in the space settings.
- **User** creates spaces, up to the limit the admin sets.
- **Guest** is someone who registered through an invitation sent by a non-admin. Guests join spaces as editors or viewers but cannot create spaces or administer one. A server admin can promote a guest to user.

## Invitations

Space admins invite people from **Settings → Space** with a link, optionally tied to an email address, for a chosen role. Someone with an account joins the space; someone without one can register through the link if the server admin allows it.

## Linked spaces and transfers

Someone who administers two spaces can link them in **Settings → Space**. Either side's admin can unlink them again.

Editors of both linked spaces can then send money from one to the other: in the transaction dialog, pick the small spaces tab next to *Transfer*, then the destination space and accounts. Each side is entered in its own account's currency. The tab only appears when there is a linked space you can write to.

A transfer is stored in both spaces, so each space's backup keeps its side, and the server signs every version of it. That keeps the two sides honest when one space is restored:

- Restoring a space from an older backup never changes the linked space. Transfers that differ come back as **pending** transactions in the restored space, to accept or reject; pending transactions do not affect balances.
- After unlinking, importing a space elsewhere or a signature that does not check out, the transfer is **frozen**: it stays in the balances but cannot be changed, and it syncs again once the spaces are linked again.

## Limits

Server admins set the limits in **Administration → System**:

- spaces per user (counted as the spaces they administer);
- the size quota of each space, which can also be changed per space;
- how many backups each space keeps;
- whether invitations may register new users.
