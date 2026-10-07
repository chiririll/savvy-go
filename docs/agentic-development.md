# Agentic development

Project rules for AI coding agents live in plain files in the repository, so any agent can use them.

- [`AGENTS.md`](../AGENTS.md): commands, layout, rules for the whole project.
- [`internal/AGENTS.md`](../internal/AGENTS.md): backend rules.
- [`resources/ts/AGENTS.md`](../resources/ts/AGENTS.md): frontend rules.

Agents that read `AGENTS.md` (Codex, Cursor, Copilot, Gemini CLI and others) need no setup. Nested files load when the agent works in that directory.

## Onboarding an agent

- **Claude Code:** reads `CLAUDE.md`, not `AGENTS.md`. `CLAUDE.md` files are git-ignored, so create them locally: one line, `@AGENTS.md`, next to each `AGENTS.md` (root, `internal/`, `resources/ts/`).
- **Other agents:** point them at `AGENTS.md` if they do not pick it up on their own.

Agent-specific files and folders (`CLAUDE.md`, `.claude/`, `.cursor/`, `.codex/`, `.gemini/`, `.windsurf/`) are git-ignored. Keep personal settings and permission allowlists there.

Shared project skills go in `.agents/skills/<name>/SKILL.md`. It is committed, and agents that support the open Agent Skills format read it.

## Writing the rules

- Caveman style: short fragments, no filler, exact technical terms.
- Put a rule in the nested file of the domain it belongs to. Root file only for what applies everywhere.
- After a big feature, update the matching `AGENTS.md`: add new commands and decisions, delete stale lines.
- Do not copy what the code or `docs/` already say. Link instead.
