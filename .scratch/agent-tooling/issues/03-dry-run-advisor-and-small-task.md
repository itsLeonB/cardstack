# 03: Dry run: advisor and small task

**What to build:** Proof, run in T3 Code Nightly, that the small-task path and the advisor role from ticket 02 work as written. Results are recorded in this file.

**Blocked by:** 02

**Status:** ready-for-human

The maintainer runs this: it needs a T3 orchestrator thread (Sonnet, high effort), which plain Claude Code cannot provide. Use a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc". Marker changes follow the 2026-10-06 probes in ticket 01: a fresh function name that did not exist before.

Cases (from the spec's Testing Decisions):
- Case 5, advisor role: launch it, send one consult, check the verdict format, confirm the model is Opus and the effort is medium in the thread, make an edit attempt and confirm it is refused, and confirm the native advisor tool is absent in the orchestrator thread.
- Case 6, small task: a single thread, no children, implements a marker change on the feature branch, verifies, reviews, commits, and consults the advisor role once.

- [ ] Each case has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [ ] Recorded as confirmed or refuted: the project environment entry disables the native advisor tool inside a launched thread; `medium` is accepted as the effort value; the advisor can run a git diff without stalling under auto-accept-edits and plan mode.
- [ ] Every failure becomes a doc fix in ticket 02's files or a new ticket.
- [ ] The throwaway branch, the advisor thread and any leftover worktree are removed.

## Comments

## Results
