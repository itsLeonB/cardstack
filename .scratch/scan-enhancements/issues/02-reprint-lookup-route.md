# 02: Reprint lookup route

**What to build:** A signed-in caller can ask for the Reprints of the Cards the matcher offered. `GET /scan/reprints` takes up to five `cardIds` and an `offset` and returns a page of five Card summaries and `hasMore`. The rule is ADR-0023: for each id in the order given, Cards with the same normalized name (case and whitespace folded, otherwise exact), excluding every id sent, whose catalog embedding is within `MATCH_REPRINT_DELTA` (a maximum cosine distance, default 0.05) of the source Card's catalog embedding. Order: the source's position in the ids given, then closeness to the source, then newest Expansion Set first. A Card reachable from two sources appears once, at its first place. The same ids give the same order on every call. Cards with no embedding and Cards in unpublished Expansion Sets are left out.

Spec: `.scratch/scan-enhancements/spec.md` (Reprints). Decision: `docs/adr/0023-reprints-are-found-by-name-and-catalog-art-not-by-the-photo-score.md`. Vocabulary: Reprint in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Cards with the same name and near-identical catalog art in other Expansion Sets are returned; the same name with different art is not
- [ ] Name matching folds case and whitespace and is otherwise exact, so "Pikachu ex" is not "Pikachu"
- [ ] Every id sent is excluded from the result, whether or not it is a source of another id
- [ ] Results are ordered by source position, then closeness to the source, then newest Expansion Set; a Card reachable from two sources appears once, at its first place
- [ ] Pages are five Cards; `hasMore` is true exactly while more remain; the same ids and offset return the same page
- [ ] Cards with no embedding and Cards in unpublished Expansion Sets never appear
- [ ] When a Card has several embeddings, the nearest one is used for the art test
- [ ] More than five ids, a negative offset, and a malformed id are refused; unknown ids are dropped without an error
- [ ] `MATCH_REPRINT_DELTA` is a setting that must be at least 0 and at most 1, and an invalid value fails the boot
- [ ] A Guest or an invalid token is rejected as on the match route, and the route shares the match route's rate limit
- [ ] The name lookup has an index it can use
- [ ] The updated API contract is regenerated
- [ ] All of the above is covered by tests at the HTTP boundary against real Postgres, with embeddings seeded directly
