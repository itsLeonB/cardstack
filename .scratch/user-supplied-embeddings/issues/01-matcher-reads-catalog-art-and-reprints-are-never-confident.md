# 01: Matcher reads catalog art only, and a Card with a Reprint is never confident

**What to build:** Two matcher changes that must land before any user photo can enter it. Queries about the catalog itself read catalog embeddings only: the Reprint art distance, the tuning script's reports and the pending-embedding query. The nearest-neighbour search stays on every source. A match whose top Card has a Reprint is never confident, whatever the margin says, so the user still picks the print; a Card without a Reprint is matched as before.

Spec: `.scratch/user-supplied-embeddings/spec.md` (Embedding, Matcher). ADR-0027. Vocabulary: Card Embedding, User-supplied Embedding and Reprint in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The Reprint art distance (ADR-0023) ignores any embedding whose source is not `catalog`
- [ ] The tuning script's reports ignore any embedding whose source is not `catalog`
- [ ] The pending-embedding query still looks only at catalog rows
- [ ] The nearest-neighbour search still compares photos with embeddings of every source of the configured model
- [ ] A match whose top Card has a Reprint answers `confident: false`, even when the runner-up trails by more than the margin
- [ ] A match whose top Card has no Reprint is confident or not exactly as it is today
- [ ] The Reprint lookup is made only when the result would otherwise be confident
- [ ] Route tests against real Postgres cover a user-source row beside a catalog row: it does not change a Reprint list, and it does not change the pending query
