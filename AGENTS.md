# Go Savvy

Selfhosted expense tracker. Multi-currency, shared spaces. One Go process: API, React SPA, scheduler, workers. Module `savvy-go`.

## Layout

- `internal/` backend: `internal/AGENTS.md`.
- `resources/ts` frontend: `resources/ts/AGENTS.md`.
- `docs/` user and dev docs. Architecture: `docs/development.md`.
- `cmd/savvy-go` entrypoint. `scripts/` logos, screenshots. `deploy/` packaging. `.github/workflows` CI, release.

## Rules

- Multi-currency is the core feature. Pick currency-correct design over brevity. Money never float.
- New feature or bug fix: write tests. Prefer TDD, failing test first.
- Pre-release: edit `0001_initial.sql` migrations in place. No compat for old Go DBs. Laravel DBs still upgraded (`internal/legacy`).
- Each space is its own SQLite file behind `internal/store`: `docs/spaces.md`.
- User-visible change: update `docs/`.
- Do not commit, push or open PRs unless asked.
- Big feature done: update the right `AGENTS.md`. Add decisions code does not show, delete stale lines.
- Learn something new and non-obvious (gotcha, convention, decision): suggest adding it to an `AGENTS.md`. Ask first, do not edit silently.
- Keep rules domain-specific. Put each in the nested file of its domain (`internal/`, `resources/ts/`). Domain has no file yet: suggest creating one. Root only for what applies everywhere.
