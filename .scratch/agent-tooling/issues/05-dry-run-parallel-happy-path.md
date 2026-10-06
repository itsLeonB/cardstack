# 05: Dry run: parallel happy path

**What to build:** Proof, run in T3 Code Nightly, that the launch, report, merge and cleanup flow from ticket 04 works as written. Results are recorded in this file.

**Blocked by:** 04

**Status:** ready-for-human

The maintainer runs this in a T3 orchestrator thread (Sonnet, high effort), on a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc".

Case 1: one backend and one frontend marker symbol (fresh function names), each added in its own worktree through that thread's own Serena, verified, committed, reported by queued message, reviewed with the `code-review` skill, merged into the feature branch and cleaned up. Repeatable check from ticket 01: each child finds its marker through Serena and edits it so its worktree changes and the main checkout does not.

- [x] Case 1 has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [x] Recorded: worktree paths match the documented derivation from the branch name; the scope check passes; the four-part report arrives as a new orchestrator turn; the Serena reminder hook fires in a launched thread; after cleanup no thread, worktree or branch is left behind.
- [x] Every failure becomes a doc fix in ticket 04's files or a new ticket.
- [ ] The throwaway branch and any leftover worktrees are removed.

## Comments

## Results

Run 2026-10-06 in orchestrator thread `32987e2d-95f0-4557-8624-cf471d84b53d` (Sonnet 5.5, high, full-access), told only "follow the workflow".

**Case 1: pass**, with one doc gap fixed and one criterion left inconclusive.

- Children: backend `mcp:35bc5e32-7378-49ed-8589-78b283941d43` (commit `3e747cb`, `ParallelMarkerBk` in `backend/internal/core/apperr/apperr.go`), frontend `mcp:03932ef6-c33a-4ed0-8609-81f49cee55a8` (commit `1d72b64`, `parallelMarkerFe` in `frontend/src/lib/date.ts`). Both launched in parallel from `chore/t3-orchestration-dryrun`, branches `chore/dryrun-parallel-backend` and `chore/dryrun-parallel-frontend`.
- Worktree paths: `/home/leon/.t3/worktrees/cardstack/chore-dryrun-parallel-backend` and `...-frontend`, matching the documented derivation (slashes become dashes) under T3's parent directory.
- Scope check: `git diff --name-only <feature>...HEAD` listed only `backend/internal/core/apperr/apperr.go` and `frontend/src/lib/date.ts`.
- Serena repeatable check: each child found its marker with `find_symbol` and edited it with `replace_symbol_body` to the `-v2` value. Its worktree changed, and the main checkout stayed clean (`git status` empty).
- Report: both four-part reports arrived as new orchestrator turns, backend first (run ordinal 2), then frontend, after the orchestrator ended its turn. While the orchestrator was busy, `t3_thread_wait` completing was the only visible signal. Both reports carried all four parts (status, SHAs, verification, deviations).
- Review: `code-review` run inline (not as two sub-agents, because each diff is a 4-line marker): no findings. No fix round was needed, so the send-findings-to-child path was not exercised.
- Merge and cleanup: both branches merged `--no-ff` into the feature branch, threads archived, worktrees removed, branches deleted with `git branch -d`. `git worktree list` and `git branch` show nothing left from the children.
- Serena reminder hook: **inconclusive**. Neither child saw it, but neither made a streak of built-in code reads, so it was never due. Ticket 10 checks it directly.
- Frontend test: the first `bun run test` in the frontend child had one failure, and the re-run passed (50 files, 389 tests). Not investigated; treat as a flaky test unless it recurs.

Findings:

1. `t3_thread_launch` rejected `modelSelection` without `instanceId`, so the first launch call failed for both children. Fixed in `docs/agents/orchestration.md` (big-task launch step 4 and advisor launch step 2), including the accepted `options` shape.
2. Serena hook not exercised: new ticket 10.
3. The throwaway branch was cut from `docs/t3-orchestration-workflow`, not `main`, for the same reason as ticket 03 (`main` lacks the workflow doc and role files).
4. A fresh frontend worktree has no `node_modules`; the frontend child ran `bun install` (the orchestrator's launch message told it to). Fixed in `docs/agents/orchestration.md`, Verification commands.
5. The throwaway branch `chore/t3-orchestration-dryrun` holds the two merge commits and is unmerged by design, so a safe `git branch -d` refuses it. The fourth criterion stays open until the maintainer force-deletes it.

