# `/implement-spec` workflow

How a spec runs when the maintainer invokes `/implement-spec`. This is the orchestration loop: the skill `implement-spec` drives the task graph, `/implement` is not orchestration. This file holds the repo rules the skill leaves open. Everything not restated here is in `docs/agents/workflows/implement.md`.

## Implementer

One implementer per ticket, `deepseek-flash` with `max` thinking, delegated as in [Delegating](implement.md#delegating): own worktree, own branch, the [task message](implement.md#task-message) with the ticket path as the Task. A ticket spanning components gets one implementer per component, and the `backend` one goes first when the change alters `backend/openapi.json`, so the others branch from the regenerated contract.

The implementer is the worker and implements directly in its own context, including its own review ([Review](../handoff.md#review)). The worker selects the task-specific skills; a ticket is one loop, reviewed once by its implementer.

## Merger

The merger subagent runs the [scope check](implement.md#delegating) first, then merges with `merge-ticket` and removes the worktree and branch ([Merge and clean up](implement.md#delegating)).

## Integration review

The review of the integration branch (skill step 7) is one more loop: its fixes are not reviewed again, and verification reruns after them ([Verification](implement.md#verification)). Findings in component code go to a fresh implementer on a new branch from the integration branch.

## Advisor and recovery

Any agent may spawn the advisor (`docs/agents/roles/advisor.md`) at its triggers. A failed or reportless implementer follows [Failure recovery](implement.md#failure-recovery). The orchestrator confirms with the maintainer before pushing the integration branch.
