# Git conventions

## Commit naming

`<type>(<component>): <message>`

Example: `feat(backend): add users api`

## Pre-commit checks

`.githooks/pre-commit` runs the checks that are fast enough for every commit, through the same `scripts/verification/*.sh` steps CI runs (they need the compose stack): `gofmt`, `go vet` and golangci-lint for `backend/`, and oxlint, Prettier and `tsc` for each of `frontend/` and `admin/` with staged files. A change to shared infrastructure (`compose.yaml`, `compose.defaults.env`, `scripts/verification/**`, `.githooks/**`) runs both components' checks, the way CI's `paths:` filters fire for it. `frontend/tooling.Dockerfile` builds both frontend packages, so it runs both of theirs. Tests and e2e stay with CI: they need the database and minutes. Without a reachable Docker daemon it skips rather than blocks.

`core.hooksPath` lives in the shared repository config, so a worktree created with `git worktree add` already has the tracked hooks active: `git -C <worktree> config --get core.hooksPath` prints `.githooks`. A fresh clone has no config of its own and prints nothing, so `scripts/setup-environment.sh` sets it once.

## Branch naming

`<type>/<name>`

Example: `feat/implement-users-management`

The convention applies only where the caller can choose the name: the feature branch, and a worker's branch when the harness lets the orchestrator name it. A harness that names the worker's branch itself is fine, because the worktree path is the contract (see [Record the worktree path](agents/workflows/implement.md#delegating)), not the name. A fix-up worker has no branch naming requirement.
