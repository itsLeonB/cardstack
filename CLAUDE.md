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
