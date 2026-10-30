# Development orchestration workflow

How the orchestrator routes a development task to either a multi-agent worktree workflow or a direct workflow.

The orchestrator is the agent the maintainer talks to. It runs on the main checkout and is the only agent that talks to the maintainer.

## Deciding which workflow applies

A task is **big** if either is true:

- It touches more than one component (`backend`, `frontend`). The `frontend` component covers both frontend packages, `frontend/` (user app) and `admin/` (Admin app, ADR-0024).
- It touches one component, but the change is large (new subsystem, cross-cutting refactor, several files and symbols, schema + API + UI surface).

Otherwise it is **small, one component**: the orchestrator does it directly, with no delegation and no worktree.

## Feature branch first

Never commit, merge or push on a protected branch (main). Before any other step, check `git branch --show-current`: if it is protected, create and switch to the feature branch (`git switch -c <branch>`, see [branch naming convention](#branch-naming-conventions)). Everything below happens on that branch: worktrees branch from it, and "the shared feature branch" means it. A `PreToolUse` hook in `.claude/settings.json` blocks `git commit`, `git merge` and `git push` while on a protected branch.

## Verification

Run `scripts/verification/<component>.sh` for each component touched, and fix every failure before committing. A fresh worktree has no installed dependencies: run `scripts/bootstrap-worktree/<component>.sh` first.

## Big task

The orchestrator stays on the main checkout and the feature branch, delegates one component task per component touched (only those), and never edits a component directory itself during a big task. Each delegate works in its own worktree with its own Serena rooted there, from the role file `.claude/agents/<component>-agent.md`.

1. **Commit first.** Inspect the worktree (`git status`) and commit only the changes the current task requires (tasks, docs, earlier merges) on the feature branch before delegating; leave unrelated changes uncommitted. A new worktree contains only commits, so uncommitted changes never reach the delegate.
2. **Order.** Delegate all components at once.
   Exception: when the change alters `backend/openapi.json`, delegate the `backend` component first, merge its branch into the feature branch, then delegate the others from that merged state, so they regenerate from the new contract.
3. **Delegate each component** with the [task message](#task-message), in its own worktree, based on the feature branch. Where you can choose the branch name, follow [the branch naming convention](#branch-naming-conventions) as `<type>/<feature>-<component>` (example: `feat/users-management-backend`), and never nest it under the feature branch (`<feature branch>/<component>`): git cannot hold a branch and a directory of the same name. Never fetch from origin; the base is the local feature branch.
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

### Review

A **loop** is one implementation pass, started by a task or by a user input: implement, review once, fix the findings, done. Code review (the `code-review` skill, managed outside this repo, so never edit it) runs once per loop, by whoever implemented it, on that implementer's own model. The reviewer evaluates the findings, discards the ones that do not hold, and fixes the rest. Fixes made from a finding are not reviewed again, so a loop ends. A new request from the maintainer starts a new loop with its own review.

- A [small task](#small-task) is one loop, reviewed by the orchestrator.
- A big task is three loops: one per worker, reviewed by that worker before it reports (task message item 7), and one by the orchestrator on the merged feature branch (step 8).

The [advisor](#advisor) is not a reviewer and is not part of any loop.

### Failure recovery

A child needs recovery when its run ends failed or without a report. Recover in this order:

1. **Read first.** Read the worktree's git state with `git -C <path> log <feature branch>..HEAD` and `git -C <path> status`, and whatever the failed run left behind. Stop a delegate that is still running before replacing it, so two delegates never write to one worktree.
2. **Keep committed work.** Commits on the child's branch are never discarded. If the task is complete and verified there, carry on with the scope check as if the child had reported, and have the worker run the [review](#review) if it never ran.
3. **Re-delegate when work is uncommitted or missing.** Delegate to a replacement as in step 3 of the big task, in the existing worktree and branch, with the same [task message](#task-message). Open the message with a "continue from this state" prompt: the commits already on the branch, what `git status` shows, what remains of the task, and what the failed run revealed about why it failed.
4. **Stop after a second failure.** If the replacement also fails on the same task, stop and ask the maintainer. Recovery never includes editing a component directory yourself, even for a small remainder. If the maintainer abandons the task, its worktree may still hold uncommitted edits that `git worktree remove` refuses to delete, and its unmerged branch is refused by `git branch -d`: the orchestrator never uses `--force` or `-D`, so it asks the maintainer to approve the forced removal first.

The replacement is the child from then on.

### Task message

Every delegate gets the same message, in this order:

1. **Role.** Read `.claude/agents/<component>-agent.md` and follow it, then read `docs/agents/conventions/general.md` and `docs/agents/conventions/<component>.md`.
2. **Task.** The ticket path under `.scratch/`, or the task text.
3. **Workspace.** Your worktree is the repository root you start in: confirm it with `git rev-parse --show-toplevel`. Work on the branch it is on and never rename it.
4. **Scope.** The component directories you may edit (`backend/`, or `frontend/` and `admin/` for the frontend component), and the read-only files the role file lists.
5. **Verification.** `scripts/bootstrap-worktree/<component>.sh` once, then `scripts/verification/<component>.sh`.
6. **Commit.** The [commit naming convention](#commit-naming-conventions), on your own branch.
7. **Review.** Run the `code-review` skill once on your own diff before you report: fix the findings that hold, discard the ones that do not, and do not review the fixes again. Rerun the verification after fixing.
8. **Never.** Never push.
9. **Report.** The [report protocol](#report-protocol), as your final message.

### Report protocol

A delegate reports in its final message. A report has five parts:

1. **Status**: done, or blocked.
2. **Commit SHAs**: every commit on the branch since the base.
3. **Verification results**: each command and whether it passed.
4. **Review**: which code review findings were fixed and which were discarded, with the reason for each discard.
5. **Deviations or questions**: anything done differently from the task, or what is needed from the orchestrator.

A blocking question goes the same way, with the question in part 5, and the delegate then ends its run. The orchestrator answers from the ticket and the ADRs first, and asks the maintainer only when they do not cover it. It continues the same delegate with the answer.

## Small task

If the task touches a limited part of one component, the orchestrator implements it directly on the feature branch in the main checkout:

1. Implement directly, with no delegation and no worktree. Drive it test-first at agreed seams (`tdd` skill) when the task comes from the ticket under `.scratch/`. Use Serena for all code reads and edits (see `docs/agents/conventions/serena.md`), and context7 for any library docs needed. Load the stack skills the component's role file lists. A short task skips the [advisor](#advisor).
2. Run `scripts/verification/<component>.sh`.
3. Run the [review](#review) once (`code-review` skill, scoped to the diff), evaluate its findings, and fix them without reviewing the fixes again. Rerun verification after fixing, before committing.
4. Commit on the feature branch (never on a protected branch) using the [commit naming convention](#commit-naming-conventions). Confirm with the maintainer before pushing.

## Advisor

The advisor is an independent second opinion on a decision, on a stronger model, a subagent defined by the role file `.claude/agents/advisor.md`. Only the orchestrator spawns it; component agents never do and ask the orchestrator instead. The harness's built-in advisor feature is disabled for the project, so this subagent is the one advisor mechanism (`docs/security.md` has the setting).

**Consult triggers.** Exactly three:

1. Choosing between approaches.
2. A worker's question that the ticket and the ADRs do not answer.
3. A failure that survives two different fixes.

The advisor is not called to confirm finished work: that is the [review](#review). Short tasks skip it.

**Task.** The advisor sees only the message, so the message is self-contained. It tells the advisor to read `.claude/agents/advisor.md` and follow it, then states the options, the orchestrator's leaning, and what to read (the ticket under `.scratch/`, ADRs, and the branches or absolute worktree paths).

**Reuse.** Send each later consult to the same advisor while its context is fresh. Spawn a new one when its context has gone stale.

**Reply.** The reply opens with the recommended option and the reasons. The orchestrator treats it as advice, not an order: act on it. If the orchestrator's own evidence contradicts a specific claim, it surfaces the conflict to the advisor in one reconcile message with the evidence, then asks the maintainer.

## Models

| Role | Model | Set |
| --- | --- | --- |
| Orchestrator | Sonnet | by the maintainer |
| Component agents | Haiku | `model` in the role file |
| Advisor | Opus | `model` in the role file |

## Commit naming conventions

`<type>(<component>): <message>`

Example: `feat(backend): add users api`

## Branch naming conventions

`<type>/<name>`

Example: `feat/implement-users-management`

The convention applies only where the caller can choose the name: the feature branch, and a worker's branch when the harness lets the orchestrator name it. A harness that names the worker's branch itself is fine, because the worktree path is the contract (see [Big task](#big-task), step 4), not the name. A fix-up worker has no branch naming requirement.
