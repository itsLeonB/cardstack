# 05: Dry run: parallel happy path

**What to build:** Proof, run in T3 Code Nightly, that the launch, report, merge and cleanup flow from ticket 04 works as written. Results are recorded in this file.

**Blocked by:** 04

**Status:** ready-for-human

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort), on a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc".

Case 1: one backend and one frontend marker symbol (fresh function names), each added in its own worktree through that thread's own Serena, verified, committed, reported by queued message, reviewed with the `code-review` skill, merged into the feature branch and cleaned up. Repeatable check from ticket 01: each child finds its marker through Serena and edits it so its worktree changes and the main checkout does not.

- [ ] Case 1 has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [ ] Recorded: worktree paths match the documented derivation from the branch name; the scope check passes; the four-part report arrives as a new orchestrator turn; the Serena reminder hook fires in a launched thread; after cleanup no thread, worktree or branch is left behind.
- [ ] Every failure becomes a doc fix in ticket 04's files or a new ticket.
- [ ] The throwaway branch and any leftover worktrees are removed.

## Comments

## Results
