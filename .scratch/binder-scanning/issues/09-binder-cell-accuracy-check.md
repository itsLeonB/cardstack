# 09: Binder-cell accuracy check

**Category:** research

**What to build:** A re-runnable offline script and a short report that measure how well the matcher recognises cards cut from photos of real binder Pages, and recommend the cell margin. It reuses the page-scan screen's flatten-and-cut logic so that the measured cells match what users send. The report compares cell accuracy with the single-card numbers from Phase 1 ticket 07 and says whether the 90 percent top-1 and 98 percent top-5 bar holds for binder cells. If it does not, the report recommends the fallback from the Phase 1 spec (OCR of the printed card number narrowing the candidates) and a new ticket handles it.

Spec: `.scratch/binder-scanning/spec.md` (Testing Decisions, Further Notes). Sources: Phase 1 ticket 07 (`.scratch/card-scanning/issues/07-tune-the-matcher-on-the-full-catalog.md`) and ADR-0018. Run it alongside ticket 07; the two share the real phone photo set and the full embedded catalog.

**Blocked by:** 06 (scan a Page)

**Status:** ready-for-agent

- [ ] The script runs end to end over about 10 real binder Page photos (covering 3×3 and at least one other grid, with sleeves, glare and angle), each with a label of the Card in every pocket and which pockets are empty
- [ ] It prints cell top-1 and top-5 over the labelled occupied cells and states how empty pockets scored
- [ ] The report compares the numbers with Phase 1 ticket 07 and says whether the 90 and 98 percent bar is met on binder cells
- [ ] The report recommends a cell margin backed by measured accuracy at several margin values, and the report names the value to put in the page-scan constant
- [ ] The report states how often a pocket is no-good-candidate or unsure at the server's current `confident` rule
- [ ] No secret and no personal photo is committed, and output goes under `.scratch/binder-scanning/research/`

**Out of scope:** any product code change, switching the model, and server-side detection.
