# 08: First live run and publish

**What to build:** The operator uses the command against the real site and production data: reads TCG Collector's terms, unlocks Chrome, dry-runs one set and then all, applies, reviews the new sets and Cards, publishes what is right, then hosts images and embeds the new Cards. This needs a person at a browser and a production credential, so it is not for an agent.

**Blocked by:** 05, 06, 07

**Status:** ready-for-human

- [ ] TCG Collector's terms of use have been read and allow this use
- [ ] A dry run on one set (for example `S12a`) lists plausible missing Cards and matches the counts in the spec's notes
- [ ] The full dry run lists only `S8a-P`, `S8a-G` and `SM-P` as new sets, with no duplicate of the `AS4D`, `AS3D` and `AS2D` decks
- [ ] After `-apply`, the new sets are unpublished and invisible to a signed-in non-Admin, and the gap-filled Cards appear in their published sets
- [ ] The operator has checked each new set's Cards and published the right ones by SQL
- [ ] `host-images` has hosted the new Cards' images and set logos, and `embed-catalog` has run for the new sets
- [ ] The migration from ticket 01 has been run on production before the first apply (`make job`)
- [ ] Decide whether catalog admin ticket 08 (entering `SM-P` by hand) is still needed once `SM-P` has been ingested and checked
