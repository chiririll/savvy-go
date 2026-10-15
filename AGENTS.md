# Go Savvy

Selfhosted expense tracker. Multi-currency, shared spaces. One Go process: API, React SPA, scheduler, workers. Module `savvy-go`.

## Layout

- `internal/` backend, `resources/ts` frontend: each has own `AGENTS.md`, nested ones below.
- `docs/` user and dev docs. Architecture: `docs/development.md`.
- `cmd/savvy-go` entrypoint. `scripts/` logos, screenshots. `deploy/` packaging. `.github/workflows` CI, release.

## Rules

- Multi-currency is core feature. Pick currency-correct design over brevity. Money never float.
- New feature or bug fix: write tests. Prefer TDD, failing test first.
- Each space is own SQLite file behind `internal/store`: `docs/spaces.md`.
- User-visible change: update `docs/`.
- No commit, push or PR unless asked.
- Create, edit or delete `AGENTS.md`: follow `.agents/skills/edit-agents-md/SKILL.md` first (skill `edit-agents-md`), never edit direct. Big feature done or non-obvious thing learned: suggest change, ask first.
