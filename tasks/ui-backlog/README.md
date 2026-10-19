# UI backlog

Tasks from a list of UI notes, checked against the code. Each file is one task:
problem, where it is in the code, the fix and the tests to write first.

| # | Task | Size |
|---|------|------|
| 01 | [Public URL: "use this address" button](01-server-url-button.md) | S |
| 02 | [Collapse mail settings while mail is off](02-mail-settings-collapse.md) | S |
| 03 | ["Unconfirmed" badge only with mail configured](03-unconfirmed-badge-needs-mail.md) | S |
| 04 | [User → Security rename, drop Preferences, Passkeys tab](04-user-settings-security-tabs.md) | M |
| 05 | [Emoji and color for a space](05-space-emoji-and-color.md) | L |
| 06 | [Align the space members list](06-space-members-alignment.md) | S |
| 07 | [One invitation list on every screen](07-invitations-single-list.md) | M |
| 08 | [Usable default categories and tags](08-default-categories-and-tags.md) | M (needs a decision) |
| 09 | [Currency picker: search, cap, fallback notice](09-currency-picker-search.md) | M |
| 10 | [Require an account when creating a space](10-onboarding-require-account.md) | S |
| 11 | [Rework sync groups, cover them in screenshots](11-sync-group-rework.md) | L |
| 12 | [Demo data for the new features](12-demo-data-new-features.md) | M |
| 13 | [Test against the wrong modal type](13-modal-type-test.md) | M |

Status: all 13 are done in code and tests. Scenarios (`scripts/screenshots`) were written but not run here (they need the Docker or built app).

Order: 01–04, 06, 07, 09 are independent. 08 and 10 need a decision first.
13 before 11 makes the new dialog follow the rules. 12 goes last (needs 05, 08,
10, 11).

Rules for every task: failing test first, new strings in `en` and `ru`, update
`docs/` for user-visible changes, no commit unless asked (`AGENTS.md`).
