# 10: Dry run: Serena reminder hook in a launched thread

**What to build:** Evidence that the `serena-hooks remind` PreToolUse hook fires in a thread launched with `t3_thread_launch` (`docs/agents/conventions/serena.md` claims it does). Found by dry run 05 (`.scratch/agent-tooling/issues/05-dry-run-parallel-happy-path.md`): both children used only Serena on code files, so the streak that triggers the nudge never happened and the run proved nothing either way.

**Blocked by:** None.

**Status:** ready-for-human

The maintainer runs this in a T3 orchestrator thread. Launch one throwaway thread (any worktree strategy, Sonnet, medium) and ask it to read three or more `.go` files with the built-in `Read` or `cat`, one after another, and then report any hook reminder text it saw verbatim. Archive the thread afterwards.

- [ ] Result recorded under "Results": the reminder appeared (quote it), or it did not after a streak of built-in code reads.
- [ ] If it did not appear, the claim in `docs/agents/conventions/serena.md` is corrected or a hook fix is made.
- [ ] The throwaway thread is archived and any worktree or branch it used is removed.

## Comments

## Results
