# 09: Empty default on binder cells

**What to build:** A binder cell whose Detection is empty defaults to the Empty mark instead of Unidentified or a wrong Card, and its pill says so before the user confirms. The user can change it to a Card or to Unidentified from the cell's menu, as for any cell. A cell that failed to match, and a cell whose Detection offered no candidates, keep their current Unidentified default. Everything else about the pill, the menu and the difference ring is unchanged.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend). Vocabulary: Page Review, Detection in `GLOSSARY.md`.

**Blocked by:** 01 (empty detection in the match response)

**Status:** ready-for-agent

- [ ] A cell with an empty Detection starts as Empty, and its pill shows Empty
- [ ] That cell's menu still lists the marks and, as there are no candidates, no candidate group
- [ ] Changing it to Unidentified and back works, and the choice survives leaving and resuming the review
- [ ] A failed cell still defaults to Unidentified with Retry, and a cell with no candidates but no empty flag still defaults to Unidentified
- [ ] An Empty cell over a Slot that holds a Card is ringed as a change, like any other difference
- [ ] A confident or non-confident cell with candidates is unchanged
- [ ] All of the above is covered by tests on the binder page-scan screen with the generated API client mocked
