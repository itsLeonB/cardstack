# 02: Review and advisor flow

**What to build:** the review rules are written once. Code review runs once per implementation loop by whoever implemented, and once more by the orchestrator after the merge. The advisor is a separate role, consulted only by the orchestrator at decision points.

**Blocked by:** 01 (both edit the orchestration doc; the order avoids merge conflicts, not a real dependency).

**Delivery:** PR 1, branch `docs/orchestration-review-flow`, orchestrator-owned (docs, role files, repo-level agent instructions).

**Status:** done

- [x] The orchestration doc defines a loop as one implementation pass started by a task or a user input: implement, review once, fix the findings, done. A new user input starts a new loop with its own review.
- [x] A small task has one loop, reviewed by the orchestrator on its own model. A large task has three loops: one per worker, reviewed by that worker on its own model before it reports, and one by the orchestrator on the merged feature branch.
- [x] Findings from the integration review are fixed by a fresh worker or, outside the component directories, by the orchestrator. Those fixes are not reviewed again and verification reruns after them. The old "stop after two rounds" limit and the old per-child review loop are removed.
- [x] The task message gains a review item for the worker. The report protocol gains a line stating which review findings were fixed and which were discarded. The statement that a child never reviews its own work is removed.
- [x] The advisor section lists exactly three triggers: choosing between approaches, a worker question the ticket and ADRs do not answer, and a failure that survives two different fixes. It states that the advisor is not called to confirm finished work, and that short tasks skip it.
- [x] The advisor message is self-contained (the options, the orchestrator's leaning, what to read), and the reply opens with the recommended option and its reasons. If the orchestrator's evidence contradicts a specific claim it surfaces the conflict, sends one reconcile message, then asks the maintainer.
- [x] The advisor role file judges a decision, not finished work. Both component role files tell the worker to run code review once before reporting and to ask the orchestrator, never the advisor, when unsure. The repo-level agent instructions' advisor line matches.
