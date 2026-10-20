# 06: Ingest missing Cards

**What to build:** For each Expansion Set, the command reads TCG Collector's card list, works out which Cards the catalog lacks and, with `-apply`, adds them with the fields from their card pages. Cards missing from an Expansion Set we already hold become visible at once if the set is published; Cards of a newly created set stay hidden with it. Pages are fetched once into a local cache, so an apply after a dry run needs no requests, and `-refresh` refetches. The report lists what was added, skipped and failed.

**Blocked by:** 01, 04, 05

**Status:** resolved

- [ ] The card-list parser reads each Card's TCG Collector id, name, number and image (largest srcset entry) from a trimmed copy of the saved list pages, one page per set with no `?page=`
- [ ] The card-page parser reads rarity (the code in parentheses; `—` means `Tanpa Tanda`), illustrator (`—` means empty), regulation mark, category (Pokémon when HP is shown, else Trainer or Energi from the subtype line) and tags (Item, Supporter, Stadium, Pokémon Tool, Energi Dasar, Energi Spesial; other words dropped), from trimmed copies of the Pokémon, Trainer and Energy sample pages
- [ ] Card numbers reduce to digits (`119/126`, `177/126`, `001/SM-P`, `No. 201`); numeric numbers match existing Cards by integer value within the set; new local IDs are zero-padded to the set's numeric width, default 3, ignoring non-numeric IDs; one Card per number with variants ignored
- [ ] In a set that already exists, Basic Energy Cards are skipped, and a Card named Victory Symbol is skipped when the set has a Card whose local ID equals the set code; both are ingested normally in a newly created set; skips are counted in the report
- [ ] A card page is fetched only for a Card that would be inserted
- [ ] A new Card gets `created_by` and `updated_by` set to `tcgcollector`, `attributes` with only `regulationMark`, `source_image_url` set, `raw` holding the page's `<main>` HTML with scripts, styles, svg and extra whitespace removed, and is never inserted over an existing one
- [ ] A rarity code not in `rarities` is created on an apply run (named by its code) and not on a dry run; a Card that fails does not abort its set
- [ ] A cache-backed page source stores fetched HTML on disk, an apply after a dry run reads only from it, and `-refresh` (optionally with `-set`) refetches
- [ ] The report lists, per set, Cards added, Cards added to an already published set, skips, failures and Cards without an image
- [ ] A test fails if any query behind a user-facing card endpoint selects `raw`
- [ ] Run tests (real Postgres, fake page source) cover the parsers, matching with `1` versus `001`, padding, the Basic Energy and Victory Symbol skips, rarity and category and tag mapping, a new rarity code, rerun safety and the cache
- [ ] Backend verification script passes
