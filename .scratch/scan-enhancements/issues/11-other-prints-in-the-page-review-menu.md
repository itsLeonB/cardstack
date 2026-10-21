# 11: Other prints in the Page Review cell menu

**What to build:** A Page Review cell's menu gains the same "Other prints" entry as the card-scan picker, with the same first page, "Load more", empty message and retry. It is offered on confident cells too. A Reprint the user picks becomes the cell's choice, shown on its pill like any chosen Card, and the cell remembers that it came from the Reprints list and at what position, which survives leaving and resuming the review. A cell with an empty Detection or no candidates has no Cards to find Reprints of and does not offer the entry.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend). Decision: `docs/adr/0023-reprints-are-found-by-name-and-catalog-art-not-by-the-photo-score.md`. Vocabulary: Reprint, Page Review in `GLOSSARY.md`.

**Blocked by:** 02 (reprint lookup route), 10 (other prints in the card-scan picker)

**Status:** done — implemented on `feat/scan-enhancements` (merge `8892f30`), review fixes in `294165a`, not yet merged to `main`.

- [ ] Every cell that holds at least one candidate, confident or not, offers "Other prints" in its menu
- [ ] A cell with an empty Detection, or with no candidates, does not offer it
- [ ] Opening it fetches one page of five from the cell's candidate ids, and "Load more" behaves as in the card-scan picker
- [ ] Picking a Reprint makes it the cell's chosen Card, updates the pill and the difference ring, and can be changed again
- [ ] The pick, with its Reprint source and position, survives leaving and resuming the review
- [ ] Nothing is fetched until the entry is opened, and a failed fetch leaves the cell unchanged
- [ ] The existing menu groups, the pill and confirm are otherwise unchanged
- [ ] All of the above is covered by tests on the binder page-scan screen with the generated API client mocked
