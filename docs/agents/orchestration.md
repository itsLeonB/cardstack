# Development orchestration workflow

How the orchestrator routes a development task to either a multi-agent worktree workflow or a direct single-component workflow.

All work starts in a T3 thread. That thread is the orchestrator, runs on the main checkout, and is the only thread the maintainer talks to.

## Deciding which workflow applies

A task is **big/multi-component** if either is true:

- It touches both `./backend` and `./frontend`.
- It touches only one component, but the change is large (new subsystem, cross-cutting refactor, several files/symbols, schema + API + UI surface).

Otherwise it is **small, one component**: the orchestrator thread does it directly, with no child threads and no worktree.

## Feature branch first

Never commit or merge onto `main` directly. Before any other step, check `git branch --show-current`: if it is `main`, create and switch to the feature branch (`git switch -c <semantic branch>/<branch name>`, see [branch naming convention](#branch-naming-conventions)). Everything below happens on that branch: worktrees branch from it, and "the shared feature branch" means it. A `PreToolUse` hook in `.claude/settings.json` blocks `git commit` and `git merge` while on `main`.

## Big/multiple components task

1. Delegate work to `backend-agent` and/or `frontend-agent` (`.claude/agents/backend-agent.md`, `.claude/agents/frontend-agent.md`) — only the components actually touched.
2. Each subagent works in its own git worktree: `git worktree add ../cardstack-<component>-<feature> -b <branch-name>`. Branch names follow the [branch naming convention](#branch-naming-conventions). When the backend change alters `backend/openapi.json`, run the backend first, merge its branch into the feature branch, then create the frontend worktree from that merged state, so the frontend regenerates its client from the new contract.
3. Each subagent implements directly, using TDD at agreed seams (`tdd` skill) when the task comes from a spec/ticket file. `/implement` describes this same workflow but carries `disable-model-invocation`, so it refuses when a subagent calls it through the Skill tool ("reserved for explicit user invocation") — subagents follow its practices directly instead rather than delegating to it. Each subagent runs its own verification script before considering the work done: backend `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`; frontend `bun run lint`, `bun run check` (prettier; `bun run format` fixes it), `bun run typecheck`, `bun run test`, `bun run build`. It then commits on its own branch using the [commit naming convention](#commit-naming-conventions) and reports back — it does **not** run its own `code-review` pass (see step 4: that's the orchestrator's job, not something to route around by editing the `code-review` skill itself, which is managed by `npx skills` and gets overwritten on update).
4. The orchestrator runs the `code-review` skill itself, scoped to that subagent's worktree diff. The orchestrator has its own Agent/Task tool, so this runs as the skill's normal two-parallel-sub-agent review — no fallback needed. The orchestrator passes the findings back to the same subagent (continue it via `SendMessage`, or a fresh delegated task scoped to just the findings) to fix; the subagent fixes them, re-runs verification, and commits the fix on its branch.
5. Once every delegated subagent has finished (including any review-finding fixes from step 4), the orchestrator merges each worktree's branch back into the shared feature branch and removes the worktrees (`git worktree remove`).
6. The orchestrator spawns a separate, high-level architecture review subagent (via the `code-review` skill, scoped to the full feature branch diff) that focuses on cross-component integration and architecture, not on re-litigating what the component-level reviewers already checked.
7. The orchestrator evaluates that report:
   - Small findings: fix directly, then re-run the relevant verification script(s).
   - Larger findings: delegate back to the relevant implementer subagent (same worktree pattern) rather than fixing inline.
8. Commit the merge on the feature branch (never on `main`) and push — confirm with the user before pushing.

The orchestrator owns every edit outside `./backend` and `./frontend` (`docs/`, `GLOSSARY.md`, ADRs, the ticket's `Status:` line under `.scratch/`), because each component agent is scoped to its own directory.

Both component subagents reference their relevant skills/MCPs internally (context7 for library docs, plus stack-specific skills — see each agent file; they edit with the built-in tools, Serena is root-agent only). The orchestrator itself should load Serena for any direct edits it makes in step 8, and context7 for any library-specific question it needs to resolve itself.

## Small, one component task

If the task touches a limited part of one component, the orchestrator thread implements it directly on the feature branch in the main checkout:

1. Implement directly, with no `backend-agent`/`frontend-agent` delegation and no worktree. Drive it with TDD at agreed seams (`tdd` skill) when the task comes from a spec/ticket file. Use Serena for all code reads/edits (mandatory, see `docs/agents/conventions/serena.md` / `initial_instructions`), and context7 for any library docs needed. Load the stack-specific skill for the area touched (e.g. `golang-testing`, `tanstack-query`, `shadcn`) the same way the component agents would. Consult the [advisor](#advisor) at its consult triggers.
2. Run that component's verification script (the commands in step 3 of the big-task workflow above).
3. Run a review pass (`code-review` skill, scoped to the diff), evaluate its findings, and fix them. Re-run the verification script (step 2) after fixing findings, before committing.
4. Commit on the feature branch (never on `main`) using the [commit naming convention](#commit-naming-conventions). Confirm with the maintainer before pushing.

## Advisor

The advisor is an independent reviewer on a stronger model, launched as its own T3 thread from the role file `.claude/agents/advisor.md`. Only the orchestrator consults it; component agents never call it and ask the orchestrator instead. The native advisor tool is disabled for the project (`CLAUDE_CODE_DISABLE_ADVISOR_TOOL` in `.claude/settings.json`), so this thread is the one advisor mechanism.

**Consult triggers.** Consult before choosing between approaches, when stuck, before declaring multi-step work done, and when a child's question is not answered by the ticket or the ADRs. Doc, ticket, config and one-line edits go ahead without it.

**Launch.** One thread per run:

1. Call `orchestrator_capabilities` and take the Opus model identifier from its catalog, confirming `medium` is an accepted effort value. Never hard-code the identifier.
2. Call `t3_thread_launch` with no `workspaceStrategy` (the advisor runs on the main checkout), `interactionMode: "plan"`, `runtimeMode: "auto-accept-edits"` (reads and `git diff` run without stalling), and `modelSelection` set to that Opus identifier at `medium` effort. Set all of these explicitly, because omitted values inherit from the orchestrator.
3. The launch message tells the advisor to read `.claude/agents/advisor.md` and follow it, then states the question, the ticket path, and the branches or absolute worktree paths to read. The advisor uses Serena for reads in the main checkout and reads a child's worktree by absolute path.
4. Wait with `t3_thread_wait`, then read the reply with `t3_thread_read`.

**Reuse.** Send each later consult to the same thread with `t3_thread_send` (`mode: "queue"`), then wait and read as above. Archive the thread when the run ends. Start a fresh thread only when its context has gone stale.

**Reply.** The verdict is `APPROVE`, `CHANGES` with a list, or `BLOCK`, each with reasons. The orchestrator treats it as advice: act on it, or on disagreement send the advisor one reconcile message with the evidence, then ask the maintainer if the two still disagree.

## Models and effort

| Role | Model | Effort | Set |
| --- | --- | --- | --- |
| Orchestrator | Sonnet | high | by the maintainer in the thread |
| Component agents | Sonnet | medium | at launch |
| Advisor | Opus | medium | at launch |

## pi caveat

pi orchestration is not designed here. `.pi/agents/` symlinks only `backend-agent.md` and `frontend-agent.md`; `advisor.md` is deliberately left out, because the advisor exists only as a T3 thread launched by the orchestrator and a pi subagent entry would invite a call path this workflow forbids.

## Commit naming conventions

`<semantic commit>(<component>): <message>`

Example: `feat(backend): add users api`

## Branch naming conventions

`<semantic branch>/<branch name>`

Example: `feat/implement-users-management`
