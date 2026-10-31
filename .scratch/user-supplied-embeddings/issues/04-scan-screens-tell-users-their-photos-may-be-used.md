# 04: Scan screens tell users their photos may be used

**What to build:** The card scan and the Page Review each show one line telling the user that photos of corrected scans may be used to improve matching. There is no setting and no opt-out. It appears wherever a scan can end in a Correction, and nowhere else.

Spec: `.scratch/user-supplied-embeddings/spec.md` (User app). ADR-0027, ADR-0022.

**Blocked by:** None (can start immediately). Release order, not a code dependency: merge before ticket 03 ships to production.

**Status:** ready-for-agent

- [ ] The card scan screen shows the notice
- [ ] The Page Review shows the notice
- [ ] No other screen shows it
- [ ] There is no toggle or dismiss that records a choice
- [ ] Tests render both screens and assert the notice, and assert its absence elsewhere
