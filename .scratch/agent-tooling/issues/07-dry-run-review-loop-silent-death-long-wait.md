# 07: Dry run: review loop, silent death, long wait

**What to build:** Proof, run in T3 Code Nightly, that the review loop and the failure recovery from ticket 06 work as written, and that a long wait does not break the orchestrator. Results are recorded in this file.

**Blocked by:** 06

**Status:** done (idle-release follow-up in ticket 11)

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort), on a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc".

Cases:
- Case 2, review-fix loop: the orchestrator sends one finding to the same child thread; the child fixes, re-verifies, commits and reports again.
- Case 3, silent child death: tell one child to archive itself (this fails its own run with no message). With the wait timeout shortened for the test, the orchestrator reads the worktree's git state and relaunches into the same worktree; after a second induced failure it stops and asks.
- Case 4, long wait: the orchestrator waits more than 30 minutes on a child that is still working; confirm the orchestrator thread is not idle-released and can still receive the report.

- [x] Each case has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [x] Every failure becomes a doc fix in ticket 06's files or a new ticket.
- [ ] The throwaway branch and any leftover worktrees are removed. No dry-run worktree is left; the branch `chore/t3-orchestration-dryrun` is unmerged by design and needs a force delete.

## Comments

## Results

Run 2026-10-06 in orchestrator thread `abb1e5f9-6b9f-4285-bfc6-d62a9a2aa003` (Sonnet 5.5, high, full-access), started with `/implement` on this ticket ("follow the workflow"). Throwaway branch `chore/t3-orchestration-dryrun` was cut from `docs/t3-orchestration-workflow`, not `main`, for the same reason as tickets 03 and 05. Three children ran in parallel, one per case, all Sonnet medium in their own worktrees.

**Case 2, review-fix loop: pass.** Child `mcp:61d409fa-7983-4b72-a2e4-bf4a8a45467e` (`chore/dryrun-review-backend`) added `ReviewLoopMarkerRq` (commit `acb268d`) and reported done. The 4-line diff had no real finding, so the orchestrator sent one stand-in finding (return `"rq-v2"`) with `t3_thread_send` and `mode: "queue"`. The send returned `delivery: started` and began run ordinal 2 on the same thread at once. The child fixed it, ran the checks, committed `340a846` and reported again. The scope check was re-run on the new commits (only `backend/internal/core/apperr/apperr.go`), then merged, archived, worktree removed and branch deleted with `-d`. Observation: the child ran `go test ./...` but did not capture its exit code, so its "passed" rested on the absence of FAIL lines; the orchestrator re-ran the tests itself and they passed.

**Case 3, silent child death: pass.**
- First failure: child `mcp:cba12de1-d3d6-4e33-bef4-b9dfa6537a0d` (`chore/dryrun-death-frontend`) edited `frontend/src/lib/date.ts` through Serena, then archived itself. `t3_thread_wait` (180 s, shortened for the test) returned `status: "failed"` after about 55 s. The thread showed an `error` item "The provider event stream closed unexpectedly" and `archived: true`. No message was sent.
- Read first: `git log <feature>..HEAD` empty, `git status` showed the uncommitted `date.ts` edit. The thread is already terminal, so no interrupt was needed.
- Relaunch: `t3_thread_launch` with `existing_worktree` (the dead thread's worktree path and branch) worked, and the message opened with the "continue from this state" prompt. The replacement `mcp:22ac2204-9019-4906-98f9-4958532367c8` was told to archive itself too and failed the same way (`failed`, same error).
- Stop and ask: after the second failure the orchestrator made no third launch and no edit under `frontend/`, and asked the maintainer. The maintainer chose to discard.
- Cleanup: `git worktree remove` refused (uncommitted edit), as the doc says. After the maintainer's approval the worktree was removed with `--force` and the branch with `-D` (it was unmerged by design).

**Case 4, long wait: pass for the wait, inconclusive for idle release.** Child `mcp:91e85773-ee7d-40fd-be46-b9019c24fd9a` (`chore/dryrun-wait-backend`) was kept busy with four foreground until-loops against absolute epoch deadlines (17:13, 17:21, 17:29, 17:38 WIB). The orchestrator's first `t3_thread_wait` with `timeoutMs: 1800000` started at 17:05 and returned `timedOut: true` with `status: "running"` at about 17:35, so a 30-minute tool wait is not cut off by Claude Code or T3, and the child was not interrupted. The read-first check found it healthy (loop 3 done at 17:29, loop 4 running). A second wait returned `completed` at 17:39, the child's commit `6067344` was in its worktree, and it was merged and cleaned up. Not proven: the orchestrator never ended its turn, so it was never idle during the wait, and the "idle-released orchestrator still receives a queued report" path was not exercised. The reports from children A and C (and C's blocked report) were queued to this thread while it was busy. A follow-up with an idle orchestrator is ticket 11.

Findings:

1. The harness blocks a standalone `sleep N` Bash call (and `sleep N; cmd`) with "use Monitor with an until-loop". The first version of case 4 (four `sleep 580` calls) died on the first call: the child sent a correct `blocked` report, the orchestrator answered it with a queued `t3_thread_send`, and an until-loop against a clock deadline worked. This is a test-method issue, not a workflow doc issue; the blocked-report and reply path worked as written.
2. A self-archive fails the child's run with "The provider event stream closed unexpectedly", leaves the worktree and branch intact, and the worktree can be reused by `existing_worktree`. This matches the documented recovery, no doc change.
3. Gap fixed in `docs/agents/orchestration.md`, Failure recovery step 4: an abandoned task leaves a dirty worktree and an unmerged branch that the safe commands refuse, so the orchestrator asks the maintainer to approve the forced removal.
4. Idle-release not tested: new ticket 11.
5. The throwaway branch `chore/t3-orchestration-dryrun` holds two merge commits and is unmerged by design, so a safe `git branch -d` refuses it.
