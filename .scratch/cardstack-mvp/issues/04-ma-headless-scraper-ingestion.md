# 04: MA ingestion via headless-browser scraper

**What to build:** A manually-triggered CLI command that scrapes the official Pokémon Asia MA card-search pages using a headless browser (the listing is JS-rendered — no usable JSON/XHR endpoint exists) and populates the same catalog schema from ticket 03 with Indonesian Evolusi Mega data.

**Blocked by:** 03 (Card catalog schema + TCGDex (SV) ingestion)

**Status:** ready-for-agent

- [ ] A CLI command drives a headless browser against the official Pokémon Asia MA card-search pages
- [ ] Card name, number, rarity, image, and print variant are extracted per listed card
- [ ] Scraped data populates the Game/Expansion Set/Card/Card Variant schema from ticket 03, scoped to Indonesia's MA Expansion Set(s)
- [ ] Running the CLI command against a real database results in queryable MA Card and Card Variant records
- [ ] The command is manually triggered only — no scheduler or cron job
