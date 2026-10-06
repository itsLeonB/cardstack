# 11: Dry run: idle orchestrator receives a report after a long wait

**What to build:** Evidence that an orchestrator thread that has ended its turn is not idle-released while a child works for more than 30 minutes, and that the child's queued report starts a new orchestrator turn. Found by dry run 07 (`.scratch/agent-tooling/issues/07-dry-run-review-loop-silent-death-long-wait.md`): the orchestrator stayed inside a `t3_thread_wait` call the whole time, so it was never idle and the report path after a long idle stretch was not exercised.

**Blocked by:** None.

**Status:** ready-for-human

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort, `full-access`), on a throwaway branch. Launch one backend child with `t3_thread_launch` and have it busy for more than 30 minutes (four foreground `until [ "$(date +%s)" -ge <epoch> ]; do sleep 5; done` calls with the Bash `timeout` at 600000; a standalone `sleep N` is blocked by the harness), then add a marker, commit and report by queued message. The orchestrator ends its turn right after the launch, without calling `t3_thread_wait`, and the maintainer leaves the thread alone until the report arrives.

- [ ] Result recorded under "Results": the report started a new orchestrator turn after 30 or more minutes of idle time (record both timestamps), or the thread was released or the report was lost (exact symptom and thread IDs).
- [ ] If it failed, `docs/agents/orchestration.md` Wait and collect is corrected or a new ticket is opened.
- [ ] The child thread is archived and any worktree or branch it used is removed.

## Comments

## Results
