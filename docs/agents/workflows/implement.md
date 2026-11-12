# `/implement` workflow

How a development task runs when the maintainer invokes `/implement`. The agent the maintainer talks to is the orchestrator: it runs on its current checkout and is the only agent that talks to the maintainer. `docs/agents/workflows/implement-spec.md` builds on this file.

The skill `implement` holds the generic steps (test-first, typecheck, review, commit). This file holds the repo rules around them.

## Feature branch first

Never commit or merge on a protected branch (main). Before any other step, check `git branch --show-current`. On a protected branch, create and switch to the feature branch (`git switch -c <branch>`, see [branch naming](../../git.md#branch-naming)). On any other branch, that branch is the feature branch and worker branches merge into it. Everything below happens on it: worktrees branch from it, and "the shared feature branch" means it. A Pi extension (`.pi/extensions/block-commit-on-main.ts`) blocks `git commit` and `git merge` while on a protected branch.

## Small or big

A task is **big** if either is true:

- It touches more than one component (`backend`, `frontend`). The `frontend` component covers both frontend packages, `frontend/` (user app) and `admin/` (Admin app, ADR-0024).
- It touches one component, but the change is large (new subsystem, cross-cutting refactor, several files and symbols, schema + API + UI surface).

Otherwise it is **small**: the orchestrator implements it directly on the feature branch in its current checkout, with no worker and no worktree.

1. Use Serena for all code reads and edits (see `docs/agents/conventions/serena.md`), context7 for any library, SDK, CLI or platform-API question (resolve the vendor's own docs repo, e.g. `/railwayapp/docs`; fall back to a web search only when context7 has no match), and the stack skills the component's role file lists. A small task skips the advisor.
2. Run the [verification](#verification).
3. Run the [review](#review) once, fix the findings, and rerun verification.
4. Commit on the feature branch using the [commit naming](../../git.md#commit-naming). Confirm with the maintainer before pushing.

A big task is [delegated](#delegating).

## Verification

Run `scripts/verification/<component>.sh [step]` for each component touched, and fix every failure before committing. It is the single entrypoint, for agents and for CI, and it brings up the compose stack itself (`docs/agents/testing.md`). The backend steps are `build`, `vet`, `format`, `lint` and `test`. The frontend steps are `install`, `lint`, `format`, `typecheck`, `test` and `build` per package (`--package frontend` or `--package admin`), then `e2e`. With no step it runs them all. An `e2e` run rebuilds the app image; lint, typecheck and unit tests run against the mounted source and need no rebuild. A fresh worktree is bootstrapped automatically when it is created (`scripts/bootstrap-worktree.sh`), which copies the env files and starts the tooling containers.

## Review

Every loop ends in a review and closes with a report; both are defined in [Hand-off](../handoff.md). This section holds only how many loops a task makes.

- A [small](#small-or-big) task is one loop, reviewed by the orchestrator through the `code-review` skill, as the implementer.
- A big task is one loop per worker, reviewed by that worker before it reports, plus one orchestrator loop on the merged feature branch ([Delegating](#delegating), step 8). Two components make three loops; a single-component big task makes two.

The advisor (`docs/agents/roles/advisor.md`) is not a reviewer and is not part of any loop.

## Delegating

The orchestrator stays on its current checkout and the feature branch, delegates one component task per component touched (only those), and never edits a component directory itself during a big task. Each worker runs in its own worktree with its own Serena rooted there, from the role file `docs/agents/roles/<component>-agent.md`.

**Model tier:** Delegates must use a smaller/faster model than the main session:
- `deepseek-flash` with `high` thinking (default, handles most component tasks)
- Other available models via `orchestrator_capabilities` — always pick one tier below the orchestrator's model

1. **Commit first.** Inspect the worktree (`git status`) and commit only the changes the current task requires (tasks, docs, earlier merges) on the feature branch before delegating; leave unrelated changes uncommitted. A new worktree contains only commits, so uncommitted changes never reach the delegate.
2. **Order.** Delegate all components at once.
   Exception: when the change alters `backend/openapi.json`, delegate the `backend` component first, merge its branch into the feature branch, then delegate the others from that merged state, so they regenerate from the new contract.
3. **Delegate each component** with the [task message](#task-message), in its own worktree, based on the feature branch. Where you can choose the branch name, follow [branch naming](../../git.md#branch-naming) as `<type>/<feature>-<component>` (example: `feat/users-management-backend`), and never nest it under the feature branch (`<feature branch>/<component>`): git cannot hold a branch and a directory of the same name. Never fetch from origin; the base is the local feature branch.
4. **Record the worktree path.** The branch name is a request, the worktree path is the contract: some harnesses create the worker's branch and name it themselves. Take the absolute worktree path from the harness when it starts the worker, or from `git worktree list` otherwise, and read the live branch from git (`git -C <path> branch --show-current`) rather than assuming the requested name. Run `git -C <path> ...` for diffs and logs. Workers never rename their branch.
5. **Collect.** The delegate ends with a [report](#report-protocol); treat it as the signal that work is ready. A delegate that ends failed or without a report starts [recovery](#failure-recovery).
6. **Check scope.** `git -C <worktree path> diff --name-only <feature branch>...HEAD` must list only paths under the child's component directories. Run it only on a clean worktree: `git -C <worktree path> status --porcelain` must print nothing, because uncommitted edits (staged, unstaged or untracked) are invisible to the diff. If it prints anything, send the delegate back to commit or discard it before the merge. Tool restrictions do not enforce scope, so this check does. The delegate has already reviewed its own diff once (see [Review](#review)), so the orchestrator does not review it again here.
7. **Merge and clean up**, per child once it is done, in this order:
   1. Merge the child's branch into the feature branch: `merge-ticket merge --worktree <worktree path> --component <component>` (from `tools/`, see `docs/security.md`) reads the live branch from the worktree and refuses a detached HEAD, checks the child worktree is clean (step 6), checks scope against the component's directories (`frontend` covers `frontend/` and `admin/`), then runs `git merge --no-ff`. It refuses on a protected branch. `tools/README.md` lists the other forms.
   2. Remove the worktree: `git worktree remove <path>`. If it refuses because the worktree is dirty, inspect it: the work is already merged, so what remains is leftover state. Report it to the maintainer instead of using `--force`.
   3. Delete the branch with a safe delete: `git branch -d <branch>`, which refuses an unmerged branch. Never use `-D`. `merge-ticket cleanup --worktree <worktree path> [--issue <file>]` does items 2 and 3 together, never forces, and stops with the git message when either refuses.
8. **Integration review.** Once every child is merged, the orchestrator runs the `code-review` skill once on the full feature branch diff, focused on cross-component integration and architecture, not on re-litigating what the worker reviews checked. Every child's branch is deleted by then, so findings in component code go to a fresh delegate as in step 3 on a new branch from the feature branch (a fix-up worker has no branch naming requirement), merged as in step 7; the orchestrator fixes only what lies outside the component directories. These fixes are not reviewed again. Rerun the relevant [verification](#verification) after them.
9. Commit the merge on the feature branch (never on a protected branch) and push, confirming with the maintainer before pushing.

The orchestrator owns every edit outside the component directories (`docs/`, `GLOSSARY.md`, ADRs, the ticket's `Status:` line under `.scratch/`), because each component agent is scoped to its own directories. It uses context7 for any library question it resolves itself, and Serena for any code read.

### Failure recovery

A child needs recovery when its run ends failed or without a report. Recover in this order:

1. **Probe the delegate first.** A `delegate_task` call may return a failure while the delegate actually spawned and works. Before declaring failure, send a probe message (`t3_thread_send` with a simple "status?" or ping) to the child thread ID, or call `task_status` on the returned `taskId`. If the delegate responds or `task_status` shows working/result_available, the delegate is alive — continue waiting instead of starting recovery. (Standard recovery probe: `git worktree list` followed by `git -C <each worktree path> status` to locate uncommitted work in failed-delegate worktrees.)
2. **Read first.** Read the worktree's git state with `git -C <path> log <feature branch>..HEAD` and `git -C <path> status`, and whatever the failed run left behind. Stop a delegate that is still running before replacing it, so two delegates never write to one worktree.
3. **Keep committed work.** Commits on the child's branch are never discarded. If the task is complete and verified there, carry on with the scope check as if the child had reported, and have the worker run the [review](#review) if it never ran.
4. **Re-delegate when work is uncommitted or missing.** Delegate to a replacement as in step 3, in the existing worktree and branch, with the same [task message](#task-message). Open the message with a "continue from this state" prompt: the commits already on the branch, what `git status` shows, what remains of the task, and what the failed run revealed about why it failed.
5. **Stop after a second failure.** If the replacement also fails on the same task, stop and ask the maintainer. Recovery never includes editing a component directory yourself, even for a small remainder. If the maintainer abandons the task, its worktree may still hold uncommitted edits that `git worktree remove` refuses to delete, and its unmerged branch is refused by `git branch -d`: the orchestrator never uses `--force` or `-D`, so it asks the maintainer to approve the forced removal first.

The replacement is the child from then on.

### Task message

Every delegate gets the same four-line message. The role file is the single source for workspace, scope, verification and commit, and it names the review and report contract, so the message does not restate them; the one exception is the review-children line, which a harness reads before the role file.

1. **Directive.** The line `/implement`, so the worker runs the skill as if the maintainer had invoked it.
2. **Role.** Read `docs/agents/roles/<component>-agent.md` and follow it.
3. **Task.** The ticket path under `.scratch/`, or the task text.
4. **Review children.** One line saying the worker runs the review itself and may spawn its review children; a harness allows a child subagents only when its task says so.

### Report protocol

A delegate reports in its final message, in the five parts [Hand-off](../handoff.md#report) lists, with a blocking question in part 5. The orchestrator answers from the ticket and the ADRs first, asks the maintainer only when they do not cover it, and continues the same delegate with the answer.
