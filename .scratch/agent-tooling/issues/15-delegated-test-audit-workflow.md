# 15: Run a delegated test audit without hand-holding the worker

**What to build:** a delegated test-audit worker can start from the repo alone. Today the vendored `.agents/skills/test-audit/SKILL.md` links two files this checkout does not have (`docs/help/testing/writing-tests.md`, `.agents/skills/openclaw-testing/SKILL.md`) and it is vendored, so it cannot be edited; `docs/agents/roles/` has no role file for a test-only worker, and `docs/agents/workflows.md`'s vendored-skills sentence does not name `test-audit`. The first delegated run of the skill stopped with "restore the two missing references"; the retry only proceeded after the orchestrator explained the substitutes, the Serena fallback and the proof commands by hand in the task message. The workflow and the role file are the deliverable, in the shape the rest of `docs/agents/` already uses.

**Blocked by:** None (can start immediately).

**Status:** open

- [ ] `docs/agents/workflows/test-audit.md` states the repo rules around the vendored skill: the two upstream links are dead here, and the substitutes are `docs/agents/testing.md` plus the Testing sections of `docs/agents/conventions/backend.md` and `frontend.md`; the skill's inline rules (authoring gate, junk patterns, retention bar, audit and validation) are the whole policy; Serena is the first tool with the documented fallback in `docs/agents/conventions/serena.md`; the scope is the current change (diff against its base) unless the maintainer declares a whole-subsystem campaign; and the worker reports per `docs/agents/handoff.md`.
- [ ] `docs/agents/workflows.md` gains the `/test-audit` row, and its vendored-skills sentence names `test-audit` (it is vendored from `openclaw/openclaw` per `skills-lock.json`).
- [ ] `docs/agents/roles/test-agent.md` exists: a test-only worker's scope (the repo's tests and test-only seams; production code only when a finding proves a real bug, reported back), Serena first with the fallback, the per-component proof commands, commit on its own branch and never push, and the five-part report.
- [ ] The workflow file names the proof commands a test worker must run and report: targeted tests, then `scripts/verification/backend.sh` and `scripts/verification/frontend.sh --package frontend test` (plus `lint` and `typecheck` when a component changed), and for every deleted test the remaining owner of its contract.
- [ ] Proof: a fresh delegate given only the ticket and the workflow path finishes a single-change audit, returns the five-part report with a named verification command and its result, and asks nothing about missing references, Serena or scope.
- [ ] The workflow file stays short enough to be read on arrival: repo-specific rules only, with the generic procedure left in the vendored skill.

## Comments

2026-11-02, Wishlist session test audit: attempt one read the skill, conventions and ticket, then blocked on the two dead links and the absent Serena. Attempt two completed as a single-change audit only after the task message supplied the substitutions, the Serena fallback permission and the proof list; it reported no changes warranted and ran no gate, which the workflow file should make explicit is allowed only when nothing changed.
