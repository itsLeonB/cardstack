# Git conventions

## Commit naming

`<type>(<component>): <message>`

Example: `feat(backend): add users api`

## Pre-commit checks

`.githooks/pre-commit` runs the checks that are fast enough for every commit, through the same `scripts/verification/*.sh` steps CI runs (they need the compose stack): `gofmt`, `go vet` and golangci-lint for `backend/`, and oxlint, Prettier and `tsc` for each of `frontend/` and `admin/` with staged files. Tests and e2e stay with CI: they need the database and minutes. Without a reachable Docker daemon it skips rather than blocks.

A worktree or clone has no local git config of its own, so point git at the tracked hooks once per checkout: `git config core.hooksPath .githooks`. Both `scripts/bootstrap-worktree/*.sh` and `scripts/setup-environment.sh` do it for you.

## Branch naming

`<type>/<name>`

Example: `feat/implement-users-management`

The convention applies only where the caller can choose the name: the feature branch, and a worker's branch when the harness lets the orchestrator name it. A harness that names the worker's branch itself is fine, because the worktree path is the contract (see [Record the worktree path](agents/workflows/implement.md#delegating)), not the name. A fix-up worker has no branch naming requirement.
