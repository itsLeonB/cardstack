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

### Backend code conventions

These apply to every backend change, whether made by the root agent or a subagent.

- Wrap errors with `ungerr.Wrap`/`ungerr.Wrapf` at the exact first location an error originates in our own code (e.g. inside the helper that calls `rand.Read`), not in its callers. Callers of our own functions return the plain `err` unchanged, and the single Huma-level seam (`backend/internal/adapters/http/huma/errors.go`, ADR-0013) classifies and unwraps it once. Known, client-safe failures are returned as `ungerr.XxxError(...)` AppErrors instead.
- Never ignore an error with `_ =`. If it is non-blocking, log it with `logger.Error`/`logger.Errorf` (`backend/internal/core/logger`, whose `Global` is a safe no-op until `Init` runs) and carry on; otherwise return it.
- In tests, mock dependencies with mockery (`.mockery.yaml`, `make mocks`, generated into `backend/internal/mocks`) rather than hand-writing stubs or fakes. See `docs/agents/testing.md`.
