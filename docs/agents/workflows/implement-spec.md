# `/implement-spec` workflow

How a spec runs when the maintainer invokes `/implement-spec`. This is the orchestration loop: the skill `implement-spec` drives the task graph, `/implement` is not orchestration. This file holds the repo rules the skill leaves open. Everything not restated here is in `docs/agents/workflows/implement.md`.

## Implementer

One implementer per ticket, `deepseek-flash` with `max` thinking, delegated as in [Delegating](implement.md#delegating): own worktree, own branch, the [task message](implement.md#task-message) with the ticket path as the Task. A ticket spanning components gets one implementer per component, in the order of step 2.

The implementer is the worker and implements in its own context, spawning only the `/code-review` subagents. The task message opens with the `/implement` directive, so the implementer follows the `implement` skill, and carries the [review rules](implement.md#review): a ticket is one loop, reviewed once by its implementer.

## Merger

The merger subagent merges with `merge-ticket`, then removes the worktree and branch ([step 7](implement.md#delegating)). The [scope check](implement.md#delegating) (step 6) comes first.

## Integration review

The review of the integration branch (skill step 7) is one more loop: its fixes are not reviewed again, and verification reruns after them ([Verification](implement.md#verification)). Findings in component code go to a fresh implementer on a new branch from the integration branch.

## Advisor and recovery

Any agent may spawn the advisor (`.claude/agents/advisor.md`) at its triggers. A failed or reportless implementer follows [Failure recovery](implement.md#failure-recovery). The orchestrator confirms with the maintainer before pushing the integration branch.
