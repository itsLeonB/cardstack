# Agent tooling

Small Go commands that make agent workflow steps deterministic. This is a separate Go module (`github.com/itsLeonB/cardstack/tools`), independent of `backend/`.

| Command | What it does | Used by |
| --- | --- | --- |
| `deny-big-output` | `PreToolUse` hook (`Bash\|Read`) that refuses a `cat` or `git diff` printing more than 20 KB, and a `Read` of a file over 40 KB without `offset` or `limit` | `.claude/settings.json`; details in `docs/security.md` |
| `merge-ticket` | `merge` checks a child worktree is clean and in scope, then runs `git merge --no-ff`; `cleanup` removes the worktree, runs `git branch -d` and marks the issue done | The orchestrator, step 8 of `docs/agents/orchestration.md` |

## Install

Requires Go (the version in `go.mod`) and `~/go/bin` on `PATH`.

```sh
cd tools
go install ./cmd/...
```

This puts both binaries in `$(go env GOBIN)`, or `~/go/bin` when `GOBIN` is unset. Check with `command -v deny-big-output merge-ticket`.

The binaries live on `PATH`, not in the repo, so every git worktree finds the same copy. Install again after a fresh clone, after `git pull` changes anything under `tools/`, and on every machine that runs agents. Until `deny-big-output` is installed, its hook errors without blocking any tool call.

## Test

```sh
cd tools
go vet ./... && go test ./...
```

The `merge-ticket` test builds throwaway git repos in a temp directory.

## Use

`deny-big-output` runs on its own through the hook. To override it for one Bash call, prefix the command with `BIG_OK=1 `. Thresholds can be changed with the `DENY_BIG_BASH_MAX` and `DENY_BIG_READ_MAX` environment variables (bytes).

`merge-ticket`, run from the feature-branch checkout:

```sh
merge-ticket merge <child-branch> --scope backend
merge-ticket cleanup <child-branch> --issue .scratch/<feature>/issues/NN-<slug>.md
```

`--scope` is required and can repeat, one per component directory. `merge` refuses on a protected branch (`main`). `cleanup` never forces: if git refuses to remove a dirty worktree or delete an unmerged branch, it stops with git's message so you can report it to the maintainer.
