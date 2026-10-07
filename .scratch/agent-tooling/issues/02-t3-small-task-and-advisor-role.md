# 02: Small task and advisor role in T3

**What to build:** A T3 thread, acting as the orchestrator, does a small single-component task directly on the feature branch in the main checkout, and can launch the advisor role (Opus, medium effort) and act on its verdict. The instructions describe exactly this, and the native advisor tool is off for the project. This is the first slice, so it also states the premise that all work starts in a T3 thread. See `.scratch/agent-tooling/spec.md`, in particular Implementation Decisions "Entry point and routing" and "Advisor".

**Blocked by:** None (can start immediately).

**Status:** done — implemented on `docs/t3-orchestration-workflow`

Work on a branch such as `docs/t3-orchestration-workflow`. Docs, one agent file and one settings key only; no backend or frontend code. The big-task section of the workflow doc is rewritten by ticket 04, so leave it as it is.

- [x] The workflow doc states that all work starts in a T3 thread, keeps the big-versus-small rule and feature-branch-first, and has a small-task section: the thread implements directly with Serena, runs verification, runs a `code-review` pass, commits on the feature branch, and confirms before pushing.
- [x] A new advisor role file defines the advisor's instructions: read-only duty, the `APPROVE` / `CHANGES` (with a list) / `BLOCK` verdict, each with reasons.
- [x] The workflow doc has an advisor recipe: launch on the main checkout in plan mode with auto-accept-edits runtime mode, Opus at medium effort (model identifier looked up through the capabilities tool, never hard-coded), Serena for reads; one thread per run, reused through queued messages and archived at the end; consult triggers (before choosing between approaches, when stuck, before declaring multi-step work done, when a child's question is not answered by the ticket or ADRs); the orchestrator treats the reply as advice, sends one reconcile message on disagreement, then asks the maintainer; component agents never call the advisor.
- [x] The workflow doc has the model and effort table: orchestrator Sonnet high (set by the maintainer in the thread), component agents Sonnet medium (set at launch), advisor Opus medium (set at launch).
- [x] Project settings gain an environment entry that disables the native advisor tool (`CLAUDE_CODE_DISABLE_ADVISOR_TOOL` = `1`), have no `advisorModel` key, and remain valid JSON.
- [x] The project instruction file's Advisor section keeps "call when unsure", says only the orchestrator consults the advisor, and points at the workflow doc for the mechanism. Its orchestration line states that all work starts in a T3 thread.
- [x] The pi agent directory is checked: either the advisor file is symlinked like the other two or the reason for leaving it out is noted in the workflow doc's pi caveat (pi orchestration itself is out of scope).
- [x] Docs state rules, not upstream issue numbers. Markdown is not hand-wrapped. The general conventions doc was read before editing, and the `writing-for-agents` skill was used for the instruction files.
