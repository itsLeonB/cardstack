# 04: Dry-run sets report from saved TCG Collector pages

**What to build:** The operator runs the new `ingest-tcgcollector` command with `-dir` pointing at a saved copy of TCG Collector's Indonesian sets page, and it prints which Expansion Sets exist, which are new, each new set's Series and the currently unpublished sets. It writes nothing. This lands the command skeleton, the page-source seam, the sets-list parser, the set-code normalisation and the matching that every later ticket builds on.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A new command with a Makefile target and a doc line, manual only; flags `-dir`, `-set CODE`, `-apply` (not wired to writes yet, dry run is the default), `-refresh` (accepted, used from ticket 06); report on stdout, logs on stderr
- [ ] A page-source interface (URL in, HTML out) with a directory-reader implementation for `-dir`
- [ ] The sets-list parser reads each set's TCG Collector id, name, code, release date, card count, logo (largest srcset entry) and the Series heading it sits under, from a trimmed copy of the saved sets page
- [ ] Codes are normalised through the alias table (`MP`→`M-P`, `SVP`→`SV-P`, `SMP`→`SM-P`, `SP`→`S-P`, `AS4Di`→`AS4D`, `AS3Di`→`AS3D`, `AS2Di`→`AS2D`) and matched on exact code against `expansion_sets`
- [ ] Series headings map by a four-row table (Mega Evolution Series to Evolusi Mega, Scarlet & Violet Series to Scarlet & Violet, Sword & Shield Series to Pedang & Perisai, Sun & Moon Series to Matahari & Bulan); an unmatched heading leaves the Series empty and is reported
- [ ] The report lists existing matched sets, sets that would be created with their Series and Card count, and every currently unpublished set with code, name, Card count, Cards without an image, whether its Series is unknown, and the `UPDATE expansion_sets SET published = true WHERE code = '…'` line
- [ ] A run test (the new ingester's run against real Postgres with the fake page source) shows aliased codes match, an unknown code is reported as new, and nothing is written
- [ ] Backend verification script passes
