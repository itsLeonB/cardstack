# 05: Create missing Expansion Sets with -apply

**What to build:** With `-apply`, the command creates every Expansion Set TCG Collector lists that the catalog lacks, unpublished, with its Series, the Indonesian locale, TCG Collector's English name and release date, and its logo. It also fills the logo of any existing Expansion Set that has none, whichever Source created it, without touching anything else on that row. Each set is written in one transaction and a rerun creates nothing twice.

**Blocked by:** 01, 02, 03, 04

**Status:** ready-for-agent

- [ ] `-apply` inserts each unmatched set with `created_by` and `updated_by` set to `tcgcollector`, `published` false, the mapped Series (empty when unknown), the Indonesian locale, name, release date and `source_logo_url`
- [ ] An existing set with an empty logo gets `source_logo_url` only; its other columns, `created_by` and `updated_by` stay as they were; a set that already has a logo is left alone
- [ ] A rerun inserts nothing and changes nothing; `-set CODE` limits the run to one set
- [ ] The report's unpublished list now includes the sets just created and says they were created in this run
- [ ] Run tests (real Postgres, fake page source) cover creation, unpublished state, Series mapping, the logo fill on an Asia-created and an Admin-updated set without changing `updated_by`, and rerun safety
- [ ] Backend verification script passes
