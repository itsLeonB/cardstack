## YAML validation

Validate YAML files with `yamllint`, not a custom Python script. Accepts a file or directory. Example: `yamllint path/to/file.yaml` or `yamllint path/to/dir`.

For each finding, evaluate before acting: fix it if it's a real issue, or exclude the rule in `.yamllint.yaml` if it's a false positive or not needed (e.g. noting why, like the GitHub Actions `on:` truthy warning).

## Markdown formatting

Never hand-wrap lines in markdown documents (no manual line breaks mid-paragraph at ~80 columns). Write each paragraph, list item, and table row as a single line; let the editor/viewer soft-wrap. Hand-wrapping breaks reflow and diffs badly.

## Agent skills

### Issue tracker

Issues live as markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-role vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Development orchestration

Multi-component or large single-component tasks route through `backend-agent`/`frontend-agent` subagents in git worktrees; small tasks proceed directly. See `docs/agents/orchestration.md` for the full workflow, commit and branch naming conventions.

### Backend test environment setup

Provisioning a local Postgres for backend tests is environment-conditional (cloud/remote agent vs. local developer), and manual catalog verification should use a small seed set, not a full scrape. See `docs/agents/testing.md`.

### Deployment

Deploy, production, or preview-environment questions (stale prod bundle, `VITE_*` env vars, failed Vercel build): `docs/agents/deployment.md`.

### Code conventions

Before writing or changing code, read `docs/agents/conventions/general.md`, plus `backend.md` for `backend/` or `frontend.md` for `frontend/`, in the same folder. Record a new convention in those files only when it passes the test in `general.md` (a rule that holds across modules), never in this file or the agent definitions.

All code reading, searching, and editing goes through Serena's symbol-level MCP tools, never the built-in file tools, unless `docs/agents/conventions/serena.md` allows an exception. That file holds the tool mapping, the fallbacks, and how to reach those tools from Claude Code and pi — read it before your first tool call on a code file.
