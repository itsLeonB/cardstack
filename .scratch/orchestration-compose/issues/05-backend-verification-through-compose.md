# 05: Backend verification through compose, and its CI

**What to build:** an agent or CI runs the whole backend check with one command, through a tooling container that has the Go toolchain, a C compiler and the pinned lint tool, against the stack's database.

**Blocked by:** 04 (The stack boots from production images).

**Delivery:** PR 2, branch `chore/compose-stack`. A backend worker writes the tooling image; the orchestrator writes the script, workflow and doc changes.

**Status:** done

- [x] A backend tooling image kept with the backend component is a glibc Go image with a C compiler (the race detector needs cgo) and the lint tool pinned to the version CI uses. It bind-mounts the repository, runs as the host user so no root-owned files appear, and keeps a long-lived process so `compose exec` works. The production backend image is untouched.
- [x] The backend verification script accepts a step name (build, vet, format, lint, test) and runs all steps in order when given none. It brings the stack up itself and starts only the tooling container for steps that need no database.
- [x] Demo: in a worktree with no host Postgres and no host lint tool, the script's full run passes, including the race-detector test run against the stack's database.
- [x] The backend CI workflow runs the same script per step, keeping its parallel jobs and exact job names. Images are built with layer caching and the Go package cache is restored from a bind-mounted cache directory.
- [x] The backend worktree bootstrap script shrinks to copying the gitignored env files and starting the stack.
- [x] The tooling worker reviews its diff once before reporting. The compose and workflow YAML pass the repo's YAML linter.
