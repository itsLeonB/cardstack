# 03: Dry run: advisor and small task

**What to build:** Proof, run in T3 Code Nightly, that the small-task path and the advisor role from ticket 02 work as written. Results are recorded in this file.

**Blocked by:** 02

**Status:** done — both cases pass, two doc fixes applied, finding 4 moved to ticket 09 (see Results)

The maintainer runs this: it needs a T3 orchestrator thread (Sonnet, high effort), which plain Claude Code cannot provide. Use a throwaway branch (`chore/t3-orchestration-dryrun`, from `main`, deleted afterwards). Tell the orchestrator only "follow the workflow doc". Marker changes follow the 2026-10-06 probes in ticket 01: a fresh function name that did not exist before.

Cases (from the spec's Testing Decisions):
- Case 5, advisor role: launch it, send one consult, check the verdict format, confirm the model is Opus and the effort is medium in the thread, make an edit attempt and confirm it is refused, and confirm the native advisor tool is absent in the orchestrator thread.
- Case 6, small task: a single thread, no children, implements a marker change on the feature branch, verifies, reviews, commits, and consults the advisor role once.

- [x] Each case has a result under "Results": pass, or fail with the exact symptom and thread IDs.
- [x] Recorded as confirmed or refuted: the project environment entry disables the native advisor tool inside a launched thread; `medium` is accepted as the effort value; the advisor can run a git diff without stalling under auto-accept-edits and plan mode.
- [x] Every failure becomes a doc fix in ticket 02's files or a new ticket.
- [x] The throwaway branch, the advisor thread and any leftover worktree are removed. Both advisor threads archived, no worktree was created, and the maintainer had `chore/t3-orchestration-dryrun` (commit `2941967`, unmerged by design) force-deleted with `git branch -D`.

## Comments

## Results

Run 2026-10-06 by the orchestrator thread `92335930-c690-46f0-a594-8d030847c456` (Sonnet 5.5, effort high, full-access, started with `/implement` plus "follow the workflow doc"). Advisor thread `mcp:5b01c210-9077-4998-80ab-546448440c49`, three runs.

**Deviation:** the throwaway branch was cut from `docs/t3-orchestration-workflow`, not `main`, because `main` does not have the new workflow doc or `advisor.md` yet.

**Case 6, small task: pass.**
- No children, no worktree. `chore/t3-orchestration-dryrun` created first. `DryRunMarkerKx47` added to `backend/internal/core/apperr/apperr.go` with Serena `insert_after_symbol`.
- `go build ./...`, `go vet ./...`, `gofmt -l .` and `go test ./...` all clean.
- Review pass ran inline on the diff instead of through the skill's two sub-agents (one-function throwaway diff): no findings.
- One advisor consult before declaring done: `APPROVE`, with reasons and what it checked. Committed as `2941967`.

**Case 5, advisor role: pass, with one caveat.**
- Launch with `t3_thread_launch`: no `workspaceStrategy`, `interactionMode: "plan"`, `runtimeMode: "auto-accept-edits"`, model `claude-opus-5-5` from `orchestrator_capabilities`, `effort: medium`. `t3_thread_configuration` on the thread returned exactly those values.
- Verdict format: first run opened with `APPROVE` on its own line, then numbered reasons. As written in the role file.
- Edit attempt: asked to create `.scratch/agent-tooling/advisor-edit-probe.txt`. Reply was `BLOCK`, file not created. Caveat: the advisor refused on its own, citing plan mode and its role file, and never called a write tool, so a harness-level denial was not observed.
- Native advisor tool absent: the orchestrator thread has no `advisor` tool and `CLAUDE_CODE_DISABLE_ADVISOR_TOOL` is `1` in its shell. The advisor thread reported the same (env var `1`, `ToolSearch` for "advisor" empty).

**Assumptions:**
- Project environment entry disables the native advisor tool inside a launched thread: **confirmed** (advisor thread, see above).
- `medium` accepted as the effort value: **confirmed**; it appears in Opus 5.5's `effort` options and the launch accepted it.
- Advisor can run `git diff` without stalling under auto-accept-edits and plan mode: **confirmed**; `git diff`, `git log`, `git status` and `go vet` ran with no approval prompt.

**Findings:**
1. Doc fix applied: in plan mode the advisor wrote its verdict to a plan file (`~/.claude/plans/...`) before giving it as its final message, which contradicts "never create files". `advisor.md` now says to write the verdict in the final message, not a plan file, and not to call `ExitPlanMode`. The stray plan file was deleted.
2. Doc fix applied: the Agent tool in Claude Code lists `advisor` (and the two component agents) as subagent types, because Claude Code reads `.claude/agents/*.md`, while the `advisor.md` description said T3 does not read agent files. The description now also says Claude Code lists the file as a subagent type and the orchestrator must never spawn it that way.
3. Note only: `git log --all -S <name>` in the advisor thread hit a `t3 checkpoint` commit (`refs/t3/orchestration-v2/checkpoints/...`) that snapshots the working tree. Harmless, but an "is this name new" check should search `main` and the feature branch, not `--all`.
4. New ticket: a harness-level write denial for the advisor was not exercised. See `.scratch/agent-tooling/issues/09-advisor-harness-write-denial-check.md`.

**Retest of finding 1, 2026-10-06:** a fresh advisor thread `mcp:33da8915-dacb-4846-9cb4-1f24edd4e571` (same launch parameters, reading the updated `advisor.md`) answered a consult with `CHANGES` in its final message. No `file_change` item, no `ExitPlanMode` call, no new file in `~/.claude/plans/`. The thread's own account: its system prompt told it to write a plan file and the new line overrode that. One sample, so confirmed but not proven. The same consult also found the Status/checkbox mismatch and the missing ticket for finding 4, both fixed above.
