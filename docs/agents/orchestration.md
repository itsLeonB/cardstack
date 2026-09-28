# Development orchestration workflow

How the orchestrator (you, in the main session) routes a development task to either a multi-agent worktree workflow or a direct single-component workflow.

## Deciding which workflow applies

A task is **big/multi-component** if either is true:

- It touches both `./backend` and `./frontend`.
- It touches only one component, but the change is large (new subsystem, cross-cutting refactor, several files/symbols, schema + API + UI surface).

Otherwise it is **small, one component** — proceed without subagents.

## Big/multiple components task

1. Delegate work to `backend-agent` and/or `frontend-agent` (`.claude/agents/backend-agent.md`, `.claude/agents/frontend-agent.md`) — only the components actually touched.
2. Each subagent works in its own git worktree: `git worktree add ../cardstack-<component>-<feature> -b <branch-name>`. Branch names follow the [branch naming convention](#branch-naming-conventions).
3. Each subagent implements directly, using TDD at agreed seams (`tdd` skill) when the task comes from a spec/ticket file. `/implement` describes this same workflow but carries `disable-model-invocation`, so it refuses when a subagent calls it through the Skill tool ("reserved for explicit user invocation") — subagents follow its practices directly instead rather than delegating to it. Either way, each subagent runs its own verification script before considering the work done: backend `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`; frontend `bun run lint`, `bun run typecheck`, `bun run test`, `bun run build`.
4. Each subagent runs its own review pass via the `code-review` skill, scoped to its worktree's diff, and gets the report back. Note: a subagent spawned via the Agent tool has no Agent/Task tool of its own to spawn the skill's two parallel sub-agents (see `code-review`'s SKILL.md step 4 for the fallback) — it performs both review passes itself, sequentially, and says so in its report.
5. Each subagent fixes the reviewer's findings once, re-runs verification, then commits on its own branch using the [commit naming convention](#commit-naming-conventions).
6. Once every delegated subagent has finished, the orchestrator merges each worktree's branch back into the shared feature branch and removes the worktrees (`git worktree remove`).
7. The orchestrator spawns a separate, high-level architecture review subagent (via the `code-review` skill, scoped to the full feature branch diff) that focuses on cross-component integration and architecture, not on re-litigating what the component-level reviewers already checked.
8. The orchestrator evaluates that report:
   - Small findings: fix directly, then re-run the relevant verification script(s).
   - Larger findings: delegate back to the relevant implementer subagent (same worktree pattern) rather than fixing inline.
9. Commit the merge on the feature branch and push — confirm with the user before pushing.

Both component subagents reference their relevant skills/MCPs internally (Serena for all code edits, context7 for library docs, plus stack-specific skills — see each agent file). The orchestrator itself should load Serena for any direct edits it makes in step 8, and context7 for any library-specific question it needs to resolve itself.

## Small, one component task

If the task touches a limited part of one component, it is small enough and justified to skip subagents:

1. Implement directly — no `backend-agent`/`frontend-agent` delegation, no worktree. Drive it with TDD at agreed seams (`tdd` skill) when the task comes from a spec/ticket file, otherwise implement directly. Use Serena for all code reads/edits (mandatory, see `serena.md` / `initial_instructions`), and context7 for any library docs needed. Load the stack-specific skill for the area touched (e.g. `golang-testing`, `tanstack-query`, `shadcn`) the same way the component agents would.
2. Run that component's verification script (same commands as step 3 above).
3. Run a review pass (`code-review` skill, scoped to the diff), evaluate its findings, and fix them. The orchestrator does have an Agent/Task tool, so this runs as the skill's normal two-parallel-sub-agent review, not the subagent fallback described in step 4 above.
4. Commit using the [commit naming convention](#commit-naming-conventions) and push — confirm with the user before pushing.

## Commit naming conventions

`<semantic commit>(<component>): <message>`

Example: `feat(backend): add users api`

## Branch naming conventions

`<semantic branch>/<branch name>`

Example: `feat/implement-users-management`
