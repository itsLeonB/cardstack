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

## Verification commands

Run from the component directory, and fix every failure before committing.

- Backend: `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`.
- Frontend: `bun install` first in a fresh worktree (it has no `node_modules`), then `bun run lint`, `bun run check` (prettier; `bun run format` fixes it), `bun run typecheck`, `bun run test`, `bun run build`.

## Big/multiple components task

The orchestrator stays on the main checkout and the feature branch, launches one child thread per component touched (only those), and never edits `./backend` or `./frontend` itself during a big task. Each child is a top-level T3 thread in its own worktree with its own Serena rooted there, launched from the role files `.claude/agents/backend-agent.md` and `.claude/agents/frontend-agent.md`. Use `t3_thread_launch` for component work, never `delegate_task`: a delegated child cannot be bound to a worktree. The orchestrator thread must run in `full-access` runtime mode: launching requires a full-access (or default) calling thread, and a message's target cannot have broader permissions than its sender, so a weaker orchestrator can neither launch children nor send to them.

1. **Commit first.** Commit everything on the feature branch (tickets, docs, earlier merges) before launching. A new worktree contains only commits, so uncommitted changes never reach the child.
2. **Order by contract.** When the backend change alters `backend/openapi.json`, launch the backend first, merge its branch into the feature branch, then launch the frontend from that merged state, so the frontend regenerates its client from the new contract. Otherwise launch both at once.
3. **Look up identifiers.** Call `orchestrator_capabilities` and take the Sonnet model identifier from its catalog, confirming `medium` is an accepted value of the model's `effort` option, plus your own thread ID, which is the `parentThreadId` field of its result (not a `t3_thread_launch` parameter: it goes into the launch message). Never hard-code either.
4. **Launch each child** with `t3_thread_launch`, setting every value explicitly because omitted ones inherit from the orchestrator:
   - `workspaceStrategy`: `{type: "worktree", baseRef: "<feature branch>", branch: "<type>/<feature>-<component>", startFromOrigin: false}`. The branch follows the [branch naming convention](#branch-naming-conventions), with the component appended after a dash (`feat/users-management-backend`). Never nest it under the feature branch (`<feature branch>/<component>`): git cannot hold a branch and a directory of the same name. Never fetch from origin; the base is the local feature branch.
   - `modelSelection`: `instanceId` is the provider instance the model came from in step 3 (`claudeAgent`; the call is rejected without it), `model` is the Sonnet identifier from step 3, and `options` sets the catalog's `effort` option to `medium` (`[{"id": "effort", "value": "medium"}]`). `runtimeMode`: `full-access`, so the child runs builds, tests and commits unattended.
   - `message`: the [launch message](#launch-message).
5. **Find the worktree path.** T3 chooses it after the launch call, so the launch message carries the branch and no path. The directory name is the branch name with slashes replaced by dashes, but the parent directory is T3's, so read the absolute path with `t3_worktree_list` (pass the child's `threadId`) before the first git command against it, then run `git -C <path> ...` for diffs and logs without asking T3.
6. **Wait and collect.** Wait on each child with `t3_thread_wait` and a 30-minute `timeoutMs` per child turn (a shorter value may be used for testing), in addition to its own report: T3 sends no completion signal for launched threads, and a child can end failed without any message. A report arrives as a queued message and starts a new orchestrator turn when you are idle. Treat the [report](#report-protocol) as the signal that work is ready. A failed run starts [recovery](#failure-recovery). An expired wait does not interrupt the child, so it starts only the read-first check there.
7. **Check scope.** Diff the changed file names against the component directory: `git -C <worktree path> diff --name-only <feature branch>...HEAD` must list only paths under `backend/` or only under `frontend/`. Run it only on a clean worktree: `git -C <worktree path> status --porcelain` must print nothing, because uncommitted edits (staged, unstaged or untracked) are invisible to the diff. If it prints anything, send the child back to commit or discard it before the review. Tool restrictions no longer enforce scope, so this check does.
8. **Review.** Run the [review loop](#review-loop) until the child's diff has no finding left to fix.
9. **Merge and clean up**, per child once it is done, in this order, because T3 has no tool for removing worktrees or branches:
   1. Merge the child's branch into the feature branch.
   2. Archive the thread: `t3_thread_organize` with `action: "archive"` and the child's `threadId`.
   3. Remove the worktree: `git worktree remove <path>`. If it refuses because the worktree is dirty, inspect it: the work is already merged, so what remains is leftover state. Report it to the maintainer instead of using `--force`.
   4. Delete the branch with a safe delete: `git branch -d <branch>`, which refuses an unmerged branch. Never use `-D`.
10. **Architecture review.** Once every child is merged, the orchestrator runs the `code-review` skill scoped to the full feature branch diff, focused on cross-component integration and architecture, not on re-litigating what the component-level reviews checked. Every child is archived and its branch deleted by then, so findings in component code go to a fresh child launched as in step 4 on a new branch from the feature branch and merged as in step 9; the orchestrator fixes only what lies outside the component directories, then re-runs the relevant [verification commands](#verification-commands).
11. Commit the merge on the feature branch (never on `main`) and push, confirming with the maintainer before pushing.

The orchestrator owns every edit outside `./backend` and `./frontend` (`docs/`, `GLOSSARY.md`, ADRs, the ticket's `Status:` line under `.scratch/`), because each component agent is scoped to its own directory. It uses context7 for any library question it resolves itself, and Serena for any code read.

### Review loop

The child does not run its own `code-review` pass, and the skill itself is managed by `npx skills`, so never edit it. For each child that reports done:

1. Run the `code-review` skill on the child's worktree diff and evaluate the findings, discarding the ones that do not hold.
2. Send the findings that need a fix to the same child thread with `t3_thread_send` and `mode: "queue"`. The child keeps its context, so a fix never goes to a fresh thread.
3. The child fixes, re-runs its [verification commands](#verification-commands), commits on its own branch and sends a new [report](#report-protocol).
4. Repeat from step 1 on the new commits, re-running the scope check, until no finding is left to fix. Only then merge (step 9).

### Failure recovery

A child needs recovery when its run ends failed. When the wait for its turn expires without a report, the child may still be working, so read first and recover only if the read shows it failed, dead or stuck. Recover in this order:

1. **Read first.** Read the thread with `t3_thread_read`, and the worktree's git state with `git -C <path> log <feature branch>..HEAD` and `git -C <path> status`. A thread still making progress (new messages, tool calls or commits) is healthy: wait again. Interrupt with `t3_thread_interrupt` only a thread that is failed or confirmed dead or stuck, and before relaunching, so two threads never write to one worktree.
2. **Keep committed work.** Commits on the child's branch are never discarded. If the task is complete and verified there, carry on with the review loop as if the child had reported.
3. **Relaunch when work is uncommitted or missing.** Launch a replacement as in step 4 of the big-task steps, with `workspaceStrategy` `{type: "existing_worktree", worktreePath: "<path>", branch: "<child branch>"}` in place of the new-worktree strategy, and the same model, runtime mode and [launch message](#launch-message). Open the message with a "continue from this state" prompt: the commits already on the branch, what `git status` shows, what remains of the task, and what the thread revealed about why the first run failed.
4. **Stop after a second failure.** If the replacement also fails or times out on the same task, stop and ask the maintainer. Recovery never includes editing `./backend` or `./frontend` yourself, even for a small remainder. If the maintainer abandons the task, its worktree may still hold uncommitted edits that `git worktree remove` refuses to delete, and its unmerged branch is refused by `git branch -d`: the orchestrator never uses `--force` or `-D`, so it asks the maintainer to approve the forced removal first.

The replacement is the child from then on: it receives review findings, and cleanup archives its thread and the failed one.

### Launch message

Every child gets the same message, in this order:

1. **Role.** Read `.claude/agents/<component>-agent.md` and follow it, then read `docs/agents/conventions/general.md` and `docs/agents/conventions/<component>.md`.
2. **Task.** The ticket path under `.scratch/`, or the task text.
3. **Orchestrator.** Your thread ID, for reports and questions.
4. **Workspace.** Your branch. Your worktree is the repository root you start in: confirm it with `git rev-parse --show-toplevel`.
5. **Scope.** The component directory you may edit, and the read-only files the role file lists.
6. **Verification.** The [verification commands](#verification-commands) for the component.
7. **Commit.** The [commit naming convention](#commit-naming-conventions), on your own branch.
8. **Never.** Never push, and never archive your own thread.
9. **Report.** The [report protocol](#report-protocol), by `t3_thread_send` to the orchestrator's thread with `mode: "queue"`.

### Report protocol

A child reports by `t3_thread_send` to the orchestrator's thread with `mode: "queue"`, which starts a turn on an idle orchestrator and waits behind the current turn on a busy one. A report has four parts:

1. **Status**: done, or blocked.
2. **Commit SHAs**: every commit on the branch since the base.
3. **Verification results**: each command and whether it passed.
4. **Deviations or questions**: anything done differently from the task, or what is needed from the orchestrator.

A blocking question goes the same way, with the question in part 4, and the child then ends its turn and waits for the reply. The orchestrator answers from the ticket and the ADRs first, and asks the maintainer only when they do not cover it. It replies to the child with `t3_thread_send` and `mode: "queue"`.

## Small, one component task

If the task touches a limited part of one component, the orchestrator thread implements it directly on the feature branch in the main checkout:

1. Implement directly, with no `backend-agent`/`frontend-agent` delegation and no worktree. Drive it with TDD at agreed seams (`tdd` skill) when the task comes from a spec/ticket file. Use Serena for all code reads/edits (mandatory, see `docs/agents/conventions/serena.md` / `initial_instructions`), and context7 for any library docs needed. Load the stack-specific skill for the area touched (e.g. `golang-testing`, `tanstack-query`, `shadcn`) the same way the component agents would. Consult the [advisor](#advisor) at its consult triggers.
2. Run that component's [verification commands](#verification-commands).
3. Run a review pass (`code-review` skill, scoped to the diff), evaluate its findings, and fix them. Re-run the verification commands after fixing findings, before committing.
4. Commit on the feature branch (never on `main`) using the [commit naming convention](#commit-naming-conventions). Confirm with the maintainer before pushing.

## Advisor

The advisor is an independent reviewer on a stronger model, launched as its own T3 thread from the role file `.claude/agents/advisor.md`. Only the orchestrator consults it; component agents never call it and ask the orchestrator instead. The native advisor tool is disabled for the project (`CLAUDE_CODE_DISABLE_ADVISOR_TOOL` in `.claude/settings.json`), so this thread is the one advisor mechanism.

**Consult triggers.** Consult before choosing between approaches, when stuck, before declaring multi-step work done, and when a child's question is not answered by the ticket or the ADRs. Doc, ticket, config and one-line edits go ahead without it.

**Launch.** One thread per run:

1. Call `orchestrator_capabilities` and take the Opus model identifier from its catalog, confirming `medium` is an accepted effort value. Never hard-code the identifier.
2. Call `t3_thread_launch` with no `workspaceStrategy` (the advisor runs on the main checkout), `interactionMode: "plan"`, `runtimeMode: "auto-accept-edits"` (reads and `git diff` run without stalling), and `modelSelection` with `instanceId` (as in the big-task launch), `model` set to that Opus identifier and `options` setting the `effort` option to `medium`. Set all of these explicitly, because omitted values inherit from the orchestrator.
3. The launch message tells the advisor to read `.claude/agents/advisor.md` and follow it, then states the question, the ticket path, and the branches or absolute worktree paths to read. The advisor uses Serena for reads in the main checkout and reads a child's worktree by absolute path.
4. Wait with `t3_thread_wait`, then read the reply with `t3_thread_read`.

**Reuse.** Send each later consult to the same thread with `t3_thread_send` (`mode: "queue"`), then wait and read as above. Archive the thread when the run ends. Start a fresh thread only when its context has gone stale.

**Reply.** The verdict is `APPROVE`, `CHANGES` with a list, or `BLOCK`, each with reasons. The orchestrator treats it as advice: act on it, or on disagreement send the advisor one reconcile message with the evidence, then ask the maintainer if the two still disagree.

## Models and effort

| Role | Model | Effort | Set |
| --- | --- | --- | --- |
| Orchestrator | Sonnet | high | by the maintainer in the thread |
| Component agents | Sonnet | medium | at launch, in `modelSelection` |
| Advisor | Opus | medium | at launch, in `modelSelection` |

## pi caveat

pi orchestration is not designed here. `.pi/agents/` symlinks only `backend-agent.md` and `frontend-agent.md`; `advisor.md` is deliberately left out, because the advisor exists only as a T3 thread launched by the orchestrator and a pi subagent entry would invite a call path this workflow forbids.

## Commit naming conventions

`<semantic commit>(<component>): <message>`

Example: `feat(backend): add users api`

## Branch naming conventions

`<semantic branch>/<branch name>`

Example: `feat/implement-users-management`
