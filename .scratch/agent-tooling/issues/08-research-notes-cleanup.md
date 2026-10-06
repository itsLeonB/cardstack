# 08: Research notes cleanup

**What to build:** The research notes no longer point at tickets or comment sections that were never saved, and they record the upstream issues and open assumptions behind the T3 rules, so the maintainer can recheck them when T3 updates. See `.scratch/agent-tooling/spec.md`, Implementation Decisions "Docs to rewrite" and Further Notes.

**Blocked by:** None (can start immediately).

**Status:** done — implemented on `docs/t3-orchestration-workflow`

- [x] Every reference in the research notes to the "Comments (T3 Code follow-up)" section of ticket 01 and to tickets 02 to 06 of the earlier plan is removed or replaced by a pointer to this spec's tickets.
- [x] The research note records upstream issues #15136, #15135, #15173, #13490 and #15082 on `pingdotgg/t3code`, each with a one-line summary and the rule in the docs it explains.
- [x] The research note lists the unverified assumptions from the spec's Further Notes, each marked as to be confirmed by the dry-run tickets.
- [x] Ticket 01's remaining follow-up (the Serena policy doc's "open research" wording) is marked as done by the work in ticket 04.
- [x] Markdown is not hand-wrapped.
