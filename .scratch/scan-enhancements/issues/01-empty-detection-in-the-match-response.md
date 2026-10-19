# 01: Empty detection in the match response

**What to build:** The matcher can say "no card here". When the top similarity of a photo is below a new setting `MATCH_EMPTY_THRESHOLD`, the match response comes back with `empty: true`, `confident: false` and no candidates, instead of the five nearest Cards. The setting defaults to 0, which turns the rule off, so nothing changes until the operator sets a value. Everything else about the match is unchanged: the confident rule and margin, a confident result trimmed to its top candidate, and up to five candidates otherwise. No second "unidentified" threshold is added.

Spec: `.scratch/scan-enhancements/spec.md` (Match response). Vocabulary: Detection in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] With the setting at 0 the response is exactly what it is today, and carries `empty: false`
- [ ] A photo whose top similarity is below the setting returns `empty: true`, `confident: false` and an empty candidate list
- [ ] A photo whose top similarity equals the setting is not empty
- [ ] The setting must be at least 0 and below the confident threshold; a value outside that fails the boot with a message naming the setting, like the existing match settings
- [ ] The confident rule, margin and candidate count are unchanged for any photo above the empty threshold
- [ ] The setting is documented with the other match settings, including that it is provisional and tuned from real photos
- [ ] The updated API contract is regenerated so the frontend can pick up the `empty` field
- [ ] All of the above is covered by tests at the HTTP boundary against real Postgres, with a seeded catalog and a fake embedder
