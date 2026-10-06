# 10: Dry run: Serena reminder hook in a launched thread

**What to build:** Evidence that the `serena-hooks remind` PreToolUse hook fires in a thread launched with `t3_thread_launch` (`docs/agents/conventions/serena.md` claims it does). Found by dry run 05 (`.scratch/agent-tooling/issues/05-dry-run-parallel-happy-path.md`): both children used only Serena on code files, so the streak that triggers the nudge never happened and the run proved nothing either way.

**Blocked by:** None.

**Status:** done

The maintainer runs this in a T3 orchestrator thread. Launch one throwaway thread (any worktree strategy, Sonnet, medium) and ask it to read three or more `.go` files with the built-in `Read` or `cat`, one after another, and then report any hook reminder text it saw verbatim. Archive the thread afterwards.

- [x] Result recorded under "Results": the reminder appeared (quote it), or it did not after a streak of built-in code reads.
- [x] If it did not appear, the claim in `docs/agents/conventions/serena.md` is corrected or a hook fix is made. Not needed: it appeared. The doc now cites this result and describes the exact behavior.
- [x] The throwaway thread is archived and any worktree or branch it used is removed.

## Comments

## Results

Run 2026-10-06 in orchestrator thread `9719833c-06a9-4ee6-99f3-18b8e278e8a7` (Sonnet 5.5, full-access), through the `/implement` skill.

**Pass: the reminder hook fires in a launched thread.**

- Probe thread `mcp:21c85bea-df54-4145-bdf4-fe6bc6e3fe00` (Sonnet 5.5, medium, full-access), launched with `t3_thread_launch` into worktree `/home/leon/.t3/worktrees/cardstack/chore-dryrun-serena-hook` (branch `chore/dryrun-serena-hook`, from `docs/t3-orchestration-workflow`). It was told not to use Serena and to read five `.go` files in a row with the built-in `Read`.
- Reads 1 and 2 (`apperr.go`, `app_config.go`) returned normally. The third read (`clerk_config.go`) was blocked. The thread's raw tool log shows the result as:

  ```
  Error: PreToolUse:Read hook error: Too many consecutive read calls of files without using symbolic tools. You can continue using read now if needed, the counter was reset.
  ```

  The thread also reported an additional-context line next to it (quoted from its own answer; the raw log shows only the error): `PreToolUse:Read hook additional context: You were using many read calls on files recently. Consider using Serena's symbolic mcp tools instead for more targeted reads. You can continue using read now if needed, the counter was reset.`
- The retry of `clerk_config.go` succeeded, and the reads of `config.go` and `db_config.go` showed no further nudge: the counter resets after each nudge.
- Cleanup: thread archived, worktree was clean and had no commits, so `git worktree remove` and `git branch -d chore/dryrun-serena-hook` both succeeded without force.

Findings:

1. The streak threshold is 3 consecutive built-in code reads, and the nudge blocks that third read once (it shows as a hook error) instead of only adding context. `docs/agents/conventions/serena.md` now says so, so an agent that sees the error knows to retry or switch to Serena.
2. The hook command in `.claude/settings.json` skips the nudge when the hook input has an `agent_id` (a native subagent), so it only reaches top-level threads. That matches the claim that launched threads are covered, and this run shows it.
