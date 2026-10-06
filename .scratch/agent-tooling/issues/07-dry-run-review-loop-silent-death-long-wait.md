# 07: Dry run: review loop, silent death, long wait

**What to build:** Proof, run in T3 Code Nightly, that the review loop and the failure recovery from ticket 06 work as written, and that a long wait does not break the orchestrator. Results are recorded in this file.

**Blocked by:** 06

**Status:** ready-for-human

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort), on a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc".

Cases:
- Case 2, review-fix loop: the orchestrator sends one finding to the same child thread; the child fixes, re-verifies, commits and reports again.
- Case 3, silent child death: tell one child to archive itself (this fails its own run with no message). With the wait timeout shortened for the test, the orchestrator reads the worktree's git state and relaunches into the same worktree; after a second induced failure it stops and asks.
- Case 4, long wait: the orchestrator waits more than 30 minutes on a child that is still working; confirm the orchestrator thread is not idle-released and can still receive the report.

- [ ] Each case has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [ ] Every failure becomes a doc fix in ticket 06's files or a new ticket.
- [ ] The throwaway branch and any leftover worktrees are removed.

## Comments

## Results
