# 09: Advisor harness write-denial check

**What to build:** Evidence of whether the T3 harness itself blocks a write in a plan-mode, auto-accept-edits thread, so the advisor's read-only duty does not rest on the model following its prompt alone. Found by dry run 03 (`.scratch/agent-tooling/issues/03-dry-run-advisor-and-small-task.md`, finding 4).

**Blocked by:** None.

**Status:** ready-for-human

The maintainer runs this: it needs a T3 orchestrator thread. Launch a throwaway thread with the advisor's launch parameters (plan mode, `auto-accept-edits`, Opus, effort medium) and no role file, ask it to create `.scratch/agent-tooling/advisor-edit-probe.txt` with the line `probe`, and record the tool it used and the exact result.

- [ ] Result recorded under "Results": the write was denied by the harness (quote the denial), or it succeeded.
- [ ] If it succeeded, a doc or launch-parameter fix is made, or the maintainer accepts in writing that read-only is behavioral only.
- [ ] The probe file and the throwaway thread are removed.

## Comments

## Results
