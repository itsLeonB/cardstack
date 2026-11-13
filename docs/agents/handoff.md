# Hand-off

The contract between a delegate and the orchestrator: the review that ends every loop, and the report that carries the result back. Role files under `docs/agents/roles/` point here; this is the single source for both.

## Review

A **loop** is one implementation pass, started by a task or by a user input: implement, review once, fix the findings, done. The `code-review` skill runs once per loop, by whoever implemented it, on that implementer's own model, through the skill and never as an inline read of its own diff. The reviewer evaluates the findings, discards the ones that do not hold, and fixes the rest.

A delegated worker runs the review itself and may spawn review children. Each review child uses the provider and model of the agent running that review: the worker's for a worker review, the orchestrator's for an integration review. Fixes made from a finding are not reviewed again, so a loop ends; a new request from the maintainer starts a new loop with its own review.

How many loops a task makes is in [the implement workflow](workflows/implement.md#review). The advisor is not a reviewer and is not part of any loop.

## Report

A delegate reports in its final message. A report has five parts:

1. **Status**: done, or blocked.
2. **Commit SHAs**: every commit on the branch since the base.
3. **Verification results**: each command and its latest result on the final changes, including failures and skipped checks. Earlier passes do not replace a later failure.
4. **Review**: which code review findings were fixed and which were discarded, with the reason for each discard.
5. **Deviations or questions**: anything done differently from the task, or what is needed from the orchestrator.

A blocking question rides in part 5, and the delegate then ends its run.
