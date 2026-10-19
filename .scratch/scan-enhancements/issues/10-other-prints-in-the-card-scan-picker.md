# 10: Other prints in the card-scan picker

**What to build:** Every Scan's candidate picker gains "Other prints". Opening it fetches the first five Reprints of the Cards the Scan holds, and "Load more" fetches the next five until none remain, then disappears. It is offered on confident Scans too, where the Scan holds one candidate and the request sends that one id. A Reprint the user picks becomes the Scan's choice, and the Scan remembers that it came from the Reprints list and at what position, so that survives a refresh. "Other prints" says plainly when there are none. Nothing is fetched until it is opened.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend). Decision: `docs/adr/0023-reprints-are-found-by-name-and-catalog-art-not-by-the-photo-score.md`. Vocabulary: Reprint, Scan in `GLOSSARY.md`.

**Blocked by:** 02 (reprint lookup route), 06 (Draft Addition keeps resolved Scans with their photos)

**Status:** ready-for-agent

- [ ] The picker of every resolved Scan, confident or not, has an "Other prints" entry
- [ ] Opening it fetches one page of five with the Scan's candidate ids; the match flow makes no extra request
- [ ] "Load more" appends the next page and is gone once `hasMore` is false
- [ ] With no Reprints the entry says so and offers nothing to pick
- [ ] A Card already among the Scan's candidates never appears, and the same Card never appears twice across pages
- [ ] Picking a Reprint makes it the Scan's chosen Card, shows it like any chosen Card, and it can be changed again
- [ ] The pick, with its Reprint source and position, survives a refresh
- [ ] A failed fetch shows a retry and leaves the Scan unchanged
- [ ] The list is a reusable piece, so the Page Review menu can use it (ticket 11)
- [ ] All of the above is covered by tests on the scan route with the generated API client mocked
