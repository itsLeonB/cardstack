# 06: Review loop and failure recovery

**What to build:** The orchestrator sends review findings back to the same child thread, and recovers from a child that fails or times out without losing its work. The instructions describe exactly this. See `.scratch/agent-tooling/spec.md`, Implementation Decisions "Reporting, questions and failure" and "Review, merge and cleanup".

**Blocked by:** 04 (both tickets edit the workflow doc).

**Status:** done — implemented on `docs/t3-orchestration-workflow`

- [x] The review loop is documented: the orchestrator runs the `code-review` skill on a child's worktree diff, sends the findings to the same child thread by queued message, and the child fixes, re-verifies, commits and reports again. A fresh thread is not used for fixes.
- [x] The failure policy is documented: on a failed child or a wait timeout, the orchestrator reads the thread and the worktree's git state first; committed work is kept; if work is uncommitted or missing it relaunches a replacement into the same worktree (existing-worktree strategy) with a "continue from this state" prompt.
- [x] After a second failure on the same task the orchestrator stops and asks the maintainer, and it never edits component code itself to recover.
- [x] The wait timeout (30 minutes per child turn) is stated once, with a note that a shorter value may be used for testing.
- [x] The rules include that a child never archives its own thread (a self-archive fails its own run), without citing an upstream issue number.
- [x] Docs state rules, not upstream issue numbers. Markdown is not hand-wrapped.
