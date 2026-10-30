## Repo rules

Validate YAML with `yamllint`, and never hand-wrap lines in markdown (one line per paragraph, list item and table row). Details: `docs/agents/repo-rules.md`.

## Agent skills

### Issue tracker

Issues live as markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-role vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Development orchestration

The agent the maintainer talks to is the orchestrator. Multi-component or large single-component tasks are delegated to `backend-agent`/`frontend-agent` in git worktrees; small tasks proceed directly. See `docs/agents/orchestration.md` for the full workflow, commit and branch naming conventions.

Code review runs once per implementation loop, by whoever implemented it. The advisor is separate: the orchestrator spawns it only to choose between approaches, to answer a worker question the ticket and ADRs leave open, or after a failure survives two different fixes, never to confirm finished work and never for short tasks. Component agents ask the orchestrator, not the advisor. The rules are in `docs/agents/orchestration.md`.

### Test environment

Agents and CI share one compose stack (database, API, apps, tooling) and one set of verification scripts, never a hand-provisioned Postgres or a host toolchain. See `docs/agents/testing.md`.

### Deployment

Deploy, production, or preview-environment questions (stale prod bundle, `VITE_*` env vars, failed Vercel build, production 401/403/429, missing images): `docs/agents/deployment.md`, which routes to a file per backend setting.

### Code conventions

Before writing or changing code, read `docs/agents/conventions/general.md`, plus `backend.md` for `backend/` or `frontend.md` for `frontend/`, in the same folder. Record a new convention in those files only when it passes the test in `general.md` (a rule that holds across modules), never in this file or the agent definitions. Every agent's code reading, searching, and editing goes through Serena's symbol-level MCP tools, never the built-in file tools, unless `docs/agents/conventions/serena.md` allows an exception. That file holds the tool mapping, the fallbacks, and how to reach those tools — read it before your first tool call on a code file.
