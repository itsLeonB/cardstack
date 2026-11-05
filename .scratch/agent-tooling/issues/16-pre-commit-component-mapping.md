# 16: Pre-commit component mapping and hook bootstrap

Type: task
Status: ready-for-agent

## What to build

A commit whose staged change touches only shared infrastructure (the compose stack, a verification script, the hooks themselves) runs the same checks CI would run for it, instead of running none. A worktree added with `git worktree add` already has the tracked hooks active, and the docs no longer claim otherwise.

## Blocked by

None (can start immediately).

## Context

The hook (a67cf2a) matches `backend/`, `frontend/` and `admin/` path prefixes only, while CI's `paths:` filters also fire on `compose.yaml`, `compose.defaults.env`, `scripts/verification/**` and the workflow files. A verification-script edit therefore commits unchecked and then changes what CI runs.

`core.hooksPath` lives in the shared repository config, so a worktree inherits it: `git -C <worktree> config --get core.hooksPath` prints `.githooks`, and a fresh clone prints nothing. The `git config core.hooksPath .githooks` calls in both `scripts/bootstrap-worktree/` scripts and the comment above them ("a worktree inherits no local git config") are wrong; `docs/git.md` repeats the claim.

## Decisions

- Static steps only: `format`, `vet`, `lint` for the backend; `lint`, `format`, `typecheck` per package for the frontend apps. No `test`, no pre-push hook. A pre-push run would have gone green on the HNSW defect that motivated this (see `.scratch/cardstack-mvp/issues/40-adapter-sql-tested-against-pgvector.md`) and duplicates CI for about a minute per backend push.
- The mapping stays inline in the hook. CI cannot call a script from `paths:` filters, so a shared mapping would mean a detect job in four workflows for no current payoff. `docs/git.md` records that the hook mirrors CI's filters.
- Shared infra runs both components: `compose.yaml`, `compose.defaults.env`, `scripts/verification/**`, `.githooks/**`.
- An unreachable Docker daemon still skips the checks rather than blocking the commit.

## Acceptance criteria

- [ ] Staging only `compose.yaml` runs the backend steps and every frontend package's steps.
- [ ] Staging only a backend file runs the backend steps and nothing else.
- [ ] Staging only a frontend or admin file runs that package's steps and nothing else.
- [ ] A path already matching a component prefix is not checked twice by the shared-infra rule.
- [ ] `scripts/bootstrap-worktree/backend.sh` and `scripts/bootstrap-worktree/frontend.sh` no longer set `core.hooksPath`, and the false comment is gone.
- [ ] `scripts/setup-environment.sh` still sets it for a fresh clone.
- [ ] `docs/git.md` states that a worktree inherits the config, a clone does not, and lists the shared-infra paths.
