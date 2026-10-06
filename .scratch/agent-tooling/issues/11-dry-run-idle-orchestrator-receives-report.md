# 11: Dry run: idle orchestrator receives a report after a long wait

**What to build:** Evidence that an orchestrator thread that has ended its turn is not idle-released while a child works for more than 30 minutes, and that the child's queued report starts a new orchestrator turn. Found by dry run 07 (`.scratch/agent-tooling/issues/07-dry-run-review-loop-silent-death-long-wait.md`): the orchestrator stayed inside a `t3_thread_wait` call the whole time, so it was never idle and the report path after a long idle stretch was not exercised.

**Blocked by:** None.

**Status:** done (throwaway branch deletion pending maintainer approval)

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort, `full-access`), on a throwaway branch. Launch one backend child with `t3_thread_launch` and have it busy for more than 30 minutes (four foreground `until [ "$(date +%s)" -ge <epoch> ]; do sleep 5; done` calls with the Bash `timeout` at 600000; a standalone `sleep N` is blocked by the harness), then add a marker, commit and report by queued message. The orchestrator ends its turn right after the launch, without calling `t3_thread_wait`, and the maintainer leaves the thread alone until the report arrives.

- [x] Result recorded under "Results": the report started a new orchestrator turn after 30 or more minutes of idle time (record both timestamps), or the thread was released or the report was lost (exact symptom and thread IDs).
- [x] If it failed (it passed, so no doc change), `docs/agents/orchestration.md` Wait and collect is corrected or a new ticket is opened.
- [x] The child thread is archived and any worktree or branch it used is removed. The child's worktree and branch `chore/dryrun-idle-report-backend` are gone. The orchestrator's throwaway branch `chore/dryrun-idle-report` holds the merge commit `f0977da`, is unmerged by design and needs a force delete (`git branch -D`) with the maintainer's approval.

## Comments

## Results

Run 2026-10-06 in orchestrator thread `4719525c-9e59-4b3c-ad2d-954493a51800` (Sonnet 5.5, `full-access`), started with `/implement` on this ticket ("follow the workflow"). Throwaway branch `chore/dryrun-idle-report` was cut from `docs/t3-orchestration-workflow`, for the same reason as tickets 03, 05 and 07. One child, `mcp:28ac52e3-6079-4a44-aebf-73a016409ccf` (Sonnet medium, branch `chore/dryrun-idle-report-backend`), was launched with `t3_thread_launch` and kept busy with four foreground until-loops against epoch deadlines (21:41, 21:49, 21:57, 22:06 WIB).

**Result: pass.** The orchestrator ended its turn right after the launch (last command at 21:29:39 WIB, no `t3_thread_wait` call) and nobody touched the thread. The child sent its queued report at 22:06:33 WIB. The report started a new orchestrator turn at 22:07 WIB (first command in it at 22:07:22), about 37 minutes after the orchestrator went idle. The thread was not idle-released and the report was not lost. The report carried all four parts (done, commit `9f17b6d`, four verification commands with exit codes, deviations). Scope check listed only `backend/internal/core/otel/otel.go`, the orchestrator re-ran the four verification commands (all passed), merged, archived the child, removed the worktree and deleted the branch with `-d`.

Findings:

1. The first until-loop's deadline (21:41) was about 644 s after the child started, over the 600000 ms Bash `timeout`, so Claude Code auto-backgrounded it. The child re-ran the loop in the foreground and returned at 21:41:03; the remaining three loops were within the limit. This is a test-method issue (the deadline was set from the orchestrator's clock, not the child's start), not a workflow doc issue. No change to `docs/agents/orchestration.md`.
2. The "Wait and collect" step already says a report starts a new orchestrator turn when the orchestrator is idle. This run is the evidence for it, so ticket 07's idle-release gap is closed.
