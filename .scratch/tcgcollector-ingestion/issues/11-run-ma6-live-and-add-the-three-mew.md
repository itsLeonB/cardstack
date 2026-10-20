# 11: Run MA6 live and add the three Mew

**What to build:** The operator re-runs the TCG Collector ingester on MA6 against the real catalog after ticket 10, so the three RGB Mew (`R/RGB`, `G/RGB`, `B/RGB`) that failed on the first run are added to the published MA6 set. This needs a person at a browser (the challenge unlock) and a production credential, so it is not for an agent.

**Blocked by:** 10

**Status:** ready-for-human

- [ ] Before the run, the production catalog has no MA6 Mew under another local ID, checked with `SELECT local_id, name FROM cards c JOIN expansion_sets s ON s.id = c.expansion_set_id WHERE s.code = 'MA6' AND c.name LIKE 'Mew%';` (a hit means Asia holds them and the apply would add a duplicate; stop and decide)
- [ ] A dry run, `go run ./cmd/ingest-tcgcollector -refresh -set=MA6`, lists the three Mew as added with `R/RGB`, `G/RGB` and `B/RGB` as their IDs, reports no failures and exits 0
- [ ] `-apply` adds them; they are visible at once, since MA6 is published, and a rerun adds nothing and counts them held
- [ ] Their rarity is as shown on the TCG Collector card pages, and a new rarity code, if any, has been created and named by its code
- [ ] `host-images` has hosted the three Cards' images, and `embed-catalog` has run for them so the scanner can match them
