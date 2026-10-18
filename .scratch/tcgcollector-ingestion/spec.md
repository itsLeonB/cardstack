# TCG Collector ingestion

Status: ready-for-agent

## Problem Statement

The catalog is filled only from the Pokémon Asia site, and that site is incomplete. Many Cards are missing from Expansion Sets the catalog already holds: against TCG Collector's Indonesian listing, `S12a` lacks about 90 Cards, `AC3a` about 100, `AC3b` about 97, `SV7s` about 57, `S-P` about 60, `SV-P` about 51 and `M-P` about 39. Whole Expansion Sets are missing too: `S8a-P`, `S8a-G` and `SM-P`. A person cannot record holdings of a Card the catalog lacks, and a Card with no hosted image cannot be matched by scanning. The operator has no way to see what is missing or to fill it without typing each Card by hand.

TCG Collector (`tcgcollector.com`) lists every one of these, but it sits behind a Cloudflare challenge, offers its API only to business partners, and writes its set names in English and a few of its set codes differently from ours.

## Solution

The operator runs a second ingestion command that reads TCG Collector's Indonesian listing and adds only what the catalog lacks: Cards missing from Expansion Sets we already hold, and Expansion Sets we do not hold at all. It never changes a row another Source wrote (ADR-0021). A new Expansion Set arrives unpublished, so nobody can record holdings against it until the operator has checked its Cards and published it. By default the command is a dry run that prints a report; `-apply` performs the writes. It reaches the site through a visible Chrome the operator unlocks once, and it works from a local page cache after that, so an apply right after a dry run makes no requests.

## User Stories

1. As the operator, I want to run one command that reads TCG Collector's Indonesian sets list, so that I can see which Expansion Sets the catalog lacks.
2. As the operator, I want the command to be a dry run unless I pass `-apply`, so that I can read what it would do before the catalog changes.
3. As the operator, I want the dry run to list every Expansion Set it would create, with its code, name, Series and Card count, so that I can spot a wrong match before applying.
4. As the operator, I want the dry run to list, per existing Expansion Set, how many Cards it would add, so that I can see how large the gap is.
5. As the operator, I want the report to list every currently unpublished Expansion Set with its code, name, Card count, Cards without an image and whether its Series is unknown, so that I am reminded what still needs checking.
6. As the operator, I want the report to print the `UPDATE` statement that publishes each unpublished set, so that I can copy it after my review.
7. As the operator, I want the report to list every Card added to an already published Expansion Set, so that I can check them, since they become visible at once.
8. As the operator, I want the report to count the Basic Energy and Victory Symbol Cards it skipped as duplicates, so that I know why a set's count differs.
9. As the operator, I want the report to list Cards that failed (unknown rarity code, unreadable page), so that I can fix the cause and rerun.
10. As the operator, I want a rerun to skip every Card and Expansion Set that already exists, so that running it twice is safe.
11. As the operator, I want `-apply` to write one transaction per Expansion Set, so that a failure in one set leaves the others complete and that set untouched.
12. As the operator, I want a `-set CODE` flag, so that I can try the command on one Expansion Set before running all 97.
13. As the operator, I want pages fetched once into a local cache, so that an apply after a dry run makes no requests and I never hit the site twice for the same page.
14. As the operator, I want a `-refresh` flag, optionally with `-set`, so that I can refetch pages when the site has changed.
15. As the operator, I want the command to open a visible Chrome with a profile that persists between runs, so that I pass the challenge by hand once and later runs start without me.
16. As the operator, I want the command to pause and wait when a challenge interstitial actually reappears and to continue when I solve it, so that a long run does not need restarting.
17. As the operator, I want requests spaced a few seconds apart, one at a time, so that I do not load the site harder than a person would.
18. As the operator, I want the command never to follow `?page=` URLs, so that it respects the site's `robots.txt`.
19. As the operator, I want a `-dir` flag that parses pages I saved by hand, so that I can ingest when no browser is available.
20. As the operator, I want the command to read the sets list once and every Card's page only for Cards it would insert, so that requests stay near the number of missing Cards.
21. As the operator, I want TCG Collector's set code normalised (`SMP` to `SM-P`, `MP` to `M-P`, `SVP` to `SV-P`, `SP` to `S-P`, `AS4Di` to `AS4D`, `AS3Di` to `AS3D`, `AS2Di` to `AS2D`) before matching, so that an existing set is never created a second time.
22. As the operator, I want a code that matches no Expansion Set to create one, so that sets we lack are added.
23. As the operator, I want a created Expansion Set to get its Series from the heading it appears under on TCG Collector (Mega Evolution Series is Evolusi Mega, Scarlet & Violet Series is Scarlet & Violet, Sword & Shield Series is Pedang & Perisai, Sun & Moon Series is Matahari & Bulan), so that it lands in the right Series.
24. As the operator, I want a created Expansion Set to use the Indonesian locale and TCG Collector's English name and release date, so that the row is complete.
25. As the operator, I want a created Expansion Set to be unpublished, so that no one sees unchecked data.
26. As a user, I want unpublished Expansion Sets to be absent from browsing, search, filters, counts, inventory pickers and card scanning, so that I never meet a Card that has not been checked.
27. As a user, I want a Card added to a published Expansion Set to appear in search straight away, so that the gap is closed without waiting.
28. As an Admin, I want to see unpublished Expansion Sets and their Cards, so that I can check them.
29. As the operator, I want the Card number on TCG Collector (`119/126`, `177/126`, `001/SM-P`, `No. 201`) reduced to its digits, so that it matches our local IDs.
30. As the operator, I want numeric Card numbers compared as integers within an Expansion Set, so that `1` and `001` are the same Card and are not inserted twice.
31. As the operator, I want a new Card's local ID zero-padded to the set's existing numeric width (3 when the set is empty), so that it sorts like the others.
32. As the operator, I want one Card per number, with the other variants ignored, so that the catalog stays one row per printed design (ADR-0006).
33. As the operator, I want a Card's name, number, image, category, tags, rarity, illustrator and regulation mark read from its page, so that the new Card is as complete as an Asia one.
34. As the operator, I want rarity taken from the code in parentheses (`Double Rare (RR)` is `RR`) and `—` mapped to `Tanpa Tanda`, so that rarities match the existing vocabulary.
35. As the operator, I want a Card with a rarity code not already in the catalog to fail into the report and not create a rarity, so that the vocabulary stays deliberate.
36. As the operator, I want category read as Pokémon, Trainer or Energi from the page and tags mapped to Item, Supporter, Stadium, Pokémon Tool, Energi Dasar and Energi Spesial, so that filters work on the new Cards.
37. As the operator, I want a Card's image URL stored as the largest webp on the page, so that the existing host-images and embed-catalog commands handle it like any other.
38. As the operator, I want each new Card to keep the stripped `<main>` HTML of its page in `raw`, so that I can read more fields later without refetching.
39. As the operator, I want Basic Energy Cards skipped in an Expansion Set that already exists, because Asia stores them under type codes such as `FIR`, so that the same card is not inserted twice.
40. As the operator, I want a Card named Victory Symbol skipped in a set that already has a Card whose local ID equals the set code, so that `M-P` and `SV-P` do not gain a duplicate.
41. As the operator, I want Basic Energy and Victory Symbol Cards ingested normally in a newly created set, so that new sets are complete.
42. As the operator, I want TCG Collector's set logo stored on any Expansion Set that has none, whoever created it, so that the logo can be shown.
43. As a user, I want an Expansion Set to show its TCG Collector logo when it has one and the Pokémon Asia cover otherwise, so that sets look right either way.
44. As the operator, I want `host-images` to rehost the logos like the other images, so that nothing is served from a third party.
45. As the operator, I want every Card and Expansion Set to record who created it and who last updated it, so that no Source overwrites another.
46. As the operator, I want the Pokémon Asia ingester to update only rows it last updated, so that it never replaces a TCG Collector or Admin row.
47. As the operator, I want an Admin's edit to take the row away from both ingesters, so that a hand correction survives every rerun.
48. As the operator, I want existing rows backfilled as created and updated by Pokémon Asia, so that nothing changes for the catalog we already hold.
49. As the operator, I want the Admin's existing behaviour unchanged except for recording `admin`, so that the catalog admin keeps working.
50. As a maintainer, I want a test that fails if a user-facing card query selects `raw`, so that the stored HTML never reaches the API.
51. As the operator, I want Cloudflare's script tags on a normal page to count as ordinary page content, so that a run never stalls asking me to solve a challenge that nobody is being shown.

## Implementation Decisions

- Two Sources already write catalog rows; this adds a third. `cards` and `expansion_sets` get `created_by` and `updated_by` text columns holding `pokemon-asia`, `tcgcollector` or `admin`. They name a Source, not a user (ADR-0021). The `curated` boolean on both tables migrates to `updated_by = admin` and is dropped; existing non-curated rows are backfilled to `pokemon-asia` in both columns.
- One rule for every writer: change a row only while `updated_by` is yourself. The Asia ingester's two upsert functions switch from "skip if curated" to this rule. The TCG Collector ingester only inserts, sets both columns to `tcgcollector`, and skips anything that already exists. The Admin write path sets `updated_by = admin` and leaves `created_by` alone.
- `expansion_sets` gets `published`, default true for existing and for Asia and Admin sets, false for sets the new ingester creates. Every user-facing catalog read (Series and set listings, name search, filters, facet counts, card detail, inventory and collection pickers, and the scan matcher) excludes unpublished sets and their Cards. Admin routes see everything. Publishing is by SQL; there is no endpoint or toggle.
- `expansion_sets` gets `source_logo_url` and `logo_key`, following the pattern of `source_image_url` and `image_key` (ADR-0016). TCG Collector fills them on any set while empty and changes no other column, including `updated_by`. `host-images` rehosts logos. Where the API returns a set's cover URL it returns the logo when `logo_key` is set and the existing cover otherwise. The logo is hosted as is, not pre-sized.
- New command and package for the ingester, mirroring the Asia one: a `cmd` entry, an ingestion adapter package, a Makefile target and a doc line. Manual only, never on Railway. Flags: `-apply`, `-set CODE`, `-dir PATH`, `-refresh`. Default is dry run. Logs and warnings go to stderr, the report to stdout.
- Pages come through a small interface (give it a URL, get HTML) with three implementations: a browser fetcher, a cache-backed fetcher that wraps it, and a directory reader for `-dir`. The browser fetcher attaches to a visible Chrome with a persistent profile, waits when a challenge interstitial actually appears, spaces requests a few seconds apart, runs one at a time, and never requests a `?page=` URL. It never uses headless mode or any technique to evade the challenge (ADR-0021), and it removes nothing from the page: Cloudflare's precursor script and its `window.__CF$cv$params` block ride on every normal page, so what marks a page as challenged is the interstitial's own config key or its `Just a moment` title, never the presence of Cloudflare's code. The cache stores raw HTML on disk keyed by URL; apply reads from it.
- The sets list gives each set's TCG Collector id, name, code, release date, card count and logo (largest srcset entry), grouped under four Series headings. A set's card list gives each Card's TCG Collector id, name, number and image (largest srcset entry) on a single page; no set needs pagination. A card page gives rarity, illustrator, regulation mark, category, tags and the Expansion row. These page shapes are in `references/tcg-collector/` and the parsers are tested against trimmed copies.
- Matching: normalise TCG Collector's set code through the alias table (`MP`, `SVP`, `SMP`, `SP`, `AS4Di`, `AS3Di`, `AS2Di`), then match on exact code. Cards match on the integer value of the number within the set. A numeric number from TCG Collector is the digits before `/` or after `No. `. New local IDs are zero-padded to the set's numeric width, default 3. Non-numeric local IDs already in a set are ignored for width.
- Duplicates by design: in an Expansion Set that already exists, skip a Card whose subtype is Basic Energy, and skip a Card named Victory Symbol when the set has a Card whose local ID equals the set code. Both are ingested normally in a newly created set. The report counts what was skipped.
- Field mapping: rarity is the code in parentheses and must already exist in `rarities`, `—` maps to `Tanpa Tanda`, and an unknown code fails the Card into the report. Category is Pokémon when the page shows HP, otherwise Trainer or Energi from the subtype line; tags come from that line and unknown words are dropped. `attributes` holds only `regulationMark`, in the key the Asia ingester already uses. Illustrator `—` is stored as empty. `raw` is the page's `<main>` element with scripts, styles, svg and extra whitespace removed.
- Series for a created set come from the heading it appears under, mapped by a four-row table. A heading that matches nothing leaves the Series empty and appears in the report.
- Each Expansion Set is written in one transaction. A Card that fails does not abort its set; it is reported and the rest are written.
- A test asserts that the queries behind user-facing card endpoints never select `raw`.

## Testing Decisions

- A good test drives the public behaviour (what rows exist after a run, what the report says, what an HTTP caller sees) and never asserts on internal helpers or call order.
- Seam one, the new ingester's run: fed by a fake page source that serves trimmed copies of the saved TCG Collector pages (sets list, `SM-P` card list, and the Pokémon, Trainer and Energy card pages), against real Postgres per ADR-0005. It covers dry run versus apply, caching, code aliases, creation of an unmatched set unpublished with the right Series, skipping existing Cards, integer matching with zero-padded storage, the Basic Energy and Victory Symbol skips, rarity and category and tag mapping, an unknown rarity failing into the report, the logo filling an empty slot without touching `updated_by`, and rerun safety.
- Seam two, the existing Pokémon Asia ingester run tests: extended so a row last updated by an Admin or TCG Collector is left alone and its own rows still update, plus the backfill result.
- Seam three, the existing catalog HTTP handler tests: extended so an unpublished set and its Cards vanish from listing, search, filters, counts and the scan matcher for a non-Admin and remain visible to an Admin, and a test for the `raw` rule.
- Prior art: the Asia ingester's run tests use a fake client and a real database; the catalog handler tests build requests against a real database.
- The browser fetcher's challenge detection is unit-tested against a trimmed copy of a normal saved page carrying Cloudflare's precursor script and its `window.__CF$cv$params` block, and against a trimmed interstitial page. The rest is not tested in CI because it needs Chrome and a live site. It is checked by hand: unlock once, dry-run one set, confirm the cache fills, and apply with the network off.
- Verification before commit follows the repo's scripts for the backend component.

## Out of Scope

- Any evasion of Cloudflare's challenge, including headless mode, stealth patches or challenge solvers.
- A scheduler or running on Railway; the command is manual.
- A publish endpoint or an Admin toggle; publishing is SQL.
- Per-Card Published state.
- Updating existing Cards from TCG Collector, or replacing an Asia row.
- Card Variants (ADR-0006), prices, and TCG Collector data beyond the fields above.
- Other regions or locales than Indonesian.
- Pre-sizing the logos to the 128px cover size, and any frontend layout change beyond checking the set list once.
- Detecting gaps from the printed total; the missing-cards CSV idea was dropped.
- Entering `SM-P` through the Admin: ticket 08 of the catalog admin should wait until `SM-P` has been ingested and checked.
- Writing embeddings; the operator runs `embed-catalog` afterwards as for any set.

## Further Notes

- The shared understanding is recorded in ADR-0021 and in the glossary entries Source, Published and Curated.
- Checked against production on 2026-10-17 (read-only): 94 Expansion Sets, none curated; every Card has a hosted image; rarity codes `ACE AR BWR C FUR K MA MUR PR R RR RRR S SAR SR SSR TR U UR` and `Tanpa Tanda`; all numeric local IDs are three digits; the Sun & Moon deck codes are `AS4D`, `AS3D`, `AS2D`; `SM-P` is not in production yet.
- TCG Collector has 97 Indonesian sets: 13 Mega Evolution, 33 Scarlet & Violet, 35 Sword & Shield, 16 Sun & Moon. Against the 94 we hold, the new ones are `S8a-P`, `S8a-G` and `SMP`.
- Sample pages are in `references/tcg-collector/`. They are 0.4 to 2.6 MB each; read slices, not whole files.
- TCG Collector's terms of use were not read, because the pages sit behind the challenge. The operator should read them before the first live run.
