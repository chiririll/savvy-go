---
name: edit-agents-md
description: Use for ANY create, edit or delete of AGENTS.md (or CLAUDE.md sibling) in this project. Check suggested change, compress, write, then audit all other AGENTS.md for redundancy and wrong scope.
---

# Editing AGENTS.md

Files load every session. Each line costs context forever. Keep short, domain-scoped, only what code cannot tell.

Steps in order. No write before step 3.

## 1. Check edit

Drop or rewrite if any answer "no":

- **Non-obvious?** Decision, reason, gotcha, convention reader of code would miss. Cut what code or tests show in minutes:
  - lists of packages, files, routes, tables, enum values, test names
  - how function works, struct fields, flags
  - rule a test already enforces (say "enforced by X" once, at most)
- **Still true?** Named file, function, flag exists. Verify before write.
- **Right scope?** Deepest directory rule applies to:
  - Root: only what applies everywhere (workflow, cross-cutting rules: money, docs, no-commit).
  - Domain (`internal/`, `resources/ts/`): domain rules.
  - Sub-domain (`internal/db/`, `internal/httpserver/`): rules that matter only in that directory.
  - No file for domain yet: create one (step 3).
- **Already said?** Search all AGENTS.md. One fact, one owner file. Update existing line, no second copy.
- **Asked?** Agents suggest AGENTS.md change, wait for approval. No silent edit unless user asked for it.

## 2. Compress

- Imperative. One fact per bullet. No preamble, no "note that", no example restating rule.
- Keep why when it is the point ("Reason: space DB may become Postgres schema"). Cut when obvious.
- Short rule over described mechanism. Name symbol only when reader needs it to find thing.
- Delete stale lines met while editing.
- File English only.

## 3. Write

- Edit owner file. Few headings. Section with one line becomes bullet.
- `CLAUDE.md` (only `@AGENTS.md`) is git-ignored, local only. Never rely on it. Tell user to create it for new `AGENTS.md` (see `docs/agentic-development.md`).
- Add or remove nested `AGENTS.md`: update `docs/agentic-development.md` list.
- Add or remove nested file: update "Nested files" pointer in parent.

## 4. Audit others

After write, read every other `AGENTS.md` (`**/AGENTS.md`). Fix same turn:

- **Redundancy:** same fact in two files, or nested file repeats parent. Keep best-scoped copy.
- **Wrong scope:** domain rule in root, or subdirectory rule in domain file. Move.
- **Too big:** file over ~40 lines, or section about one subdirectory only. Split to nested file.
- **Too thin:** nested file of 2-3 lines not tied to its directory. Fold into parent.
- **Verbose or checkable lines:** apply steps 1 and 2.
- **Dangling refs:** pointer to file or symbol that no longer exists.

Report changes in few lines. No commit.
