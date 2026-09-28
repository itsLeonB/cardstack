# Pokémon TCG Regulation Marks: verifying the "Regulasi" filter mapping

Research date: 2026-09-27. Scope: verify the claim that `asia.pokemon-card.com`'s "Regulasi" search filter (Standar / Luas / Lainnya / Semua) corresponds to official Regulation-Mark-defined tournament formats, for the cardstack data model.

Every claim below is followed by its source. Where I queried the live site directly (not just documentation), I say so explicitly — that is first-party primary evidence for how *this specific* site behaves, separate from the global (English) Play! Pokémon rules.

## 1. What a Regulation Mark is

**Confirmed.** A Regulation Mark ("Tanda Regulasi" in Indonesian) is a letter printed at the bottom-left of a card, next to/near the expansion symbol, used to control which cards are legal in official tournament play.

- Bulbapedia, *Standard format (TCG)*: "The regulation mark is a letter symbol found right next to the expansion mark of each card that identifies whether it is legal to use in tournament play." (https://bulbapedia.bulbagarden.net/wiki/Standard_format_(TCG), fetched via the site's own API: `action=parse&page=Standard_format_(TCG)`)
- Pokemon.com, *2026 Pokémon TCG Standard Format Rotation Announcement*: "A regulation mark is a symbol printed on the bottom of Pokémon trading cards that indicates which cards are legal for tournament play... regardless of which expansion it came from." (https://www.pokemon.com/us/pokemon-news/2026-pokemon-tcg-standard-format-rotation-announcement)
- **Direct confirmation on the scraped site itself** — `asia.pokemon-card.com/id/rules/regulation/` (fetched live): "Cek Tanda Regulasi yang ditampilkan di kiri bawah kartu." ("Check the Regulation Mark shown at the bottom-left of the card.") and "Tanda Regulasi adalah tanda yang ditampilkan di kiri bawah kartu dan digunakan untuk mengontrol regulasi." ("The Regulation Mark is a mark shown at the bottom-left of the card and used to control regulation.") This is the Indonesian-locale equivalent of Japan's own `pokemon-card.com/rules/regulation/` page, which says the same thing in Japanese ("カードの左下に表示された「レギュレーションマーク」").

So: bottom-left corner, next to the set symbol, governs tournament legality. Confirmed.

## 2. Standard format legality

**Not literally "the N most recent marks" as a fixed formula.** The official rule is a *floor*: a card is Standard-legal if its Regulation Mark is the current season's designated letter **or later** (open-ended at the top — any future letter is automatically legal too). The floor is raised by exactly one letter at each annual rotation. Because a new letter is typically introduced partway through a season, the *number* of concurrently-legal letters fluctuates (usually 2 right after a rotation, growing to 3 by the time the next new set/letter/rotation lands) rather than being a fixed N. Official communications express this as an enumerated list of currently-legal letters, not as "the top N," which is a secondary/community paraphrase.

- Bulbapedia's full rotation history table (fetched via API, wikitext of *Standard format (TCG)*) shows the floor-letter progression year by year:
  - 2018-19: mark A or later (Asia only)
  - 2019-20: mark B or later
  - 2020-21: mark C or later
  - 2021-23 (two seasons, no rotation in 2022-23): mark D or later, through Crown Zenith
  - 2023-24: mark E or later, through Paldean Fates
  - 2024-25: mark F or later, through Prismatic Evolutions
  - 2025-26: mark G or later, through Ascended Heroes
  - 2026-27: mark H or later
- Pokemon.com, *2026 Standard Format Rotation Announcement* (dated Jan 9, 2026 per Bulbapedia's citation of it): cards marked **H, I, or J** (and any future mark) are legal; mark **G** rotates out. Effective **April 10, 2026** for in-person Play! Pokémon events, **March 26, 2026** for Pokémon TCG Live. (https://www.pokemon.com/us/pokemon-news/2026-pokemon-tcg-standard-format-rotation-announcement)
- The scraped site's own rules page confirms the same letters in Indonesian: `asia.pokemon-card.com/id/rules/regulation/` → "Regulasi Standar" section: "Kartu yang pada Tanda Regulasi yang ditampilkan di kiri bawah kartu bertuliskan huruf **H/I/J**." Older A–G-marked cards remain deck-legal only if a currently-legal reprint of the same card exists (same rule as global Play! Pokémon).
- Japan's own official rules page (`pokemon-card.com/rules/regulation/`, fetched live) says the identical thing in Japanese: "カードの左下に表示された レギュレーションマークに、**H・I・J**と書かれたカード" are Standard-legal.

**As of the most recent dated rotation (announced Jan 9, 2026, effective March 26/April 10, 2026): H, I, J are Standard-legal; G rotated out.**

## 3. Expanded format's Regulation Mark range

**Confirmed: currently mark D onward (unchanged since the 2021-23 season), defined as a floor with no upper bound, and it has not moved since D was introduced** (unlike Standard's floor, which moves yearly). Historically, prior to Regulation Marks, Expanded was defined by set name ("Black & White onward"); once marks existed, D became a synonym for that same cutoff on the English/TPCi side, but the *set-name* framing ("Black & White onward") remains the primary official English-language definition, since D was chosen specifically to match where Expanded already began.

- Bulbapedia, *Expanded format (TCG)* (fetched via API): "Currently, the Expanded format allows use of all cards printed from Black & White onwards, minus a short ban list." Its own rotation-history table lists every season from 2014-15 through 2026-27 identically as "Black & White to [latest set], and all regulation marks" — i.e., Expanded has never rotated in practice, despite the rulebook technically allowing it to. (https://bulbapedia.bulbagarden.net/wiki/Expanded_format_(TCG))
- A community source (thecardshopfinder/PokeMasters-style aggregation, cross-checked against Bulbapedia) states the D-onward equivalence explicitly: "Expanded includes every expansion from Black & White forward, all regulation marks D through J, minus 27 banned cards." Treat this as a secondary paraphrase of the same Bulbapedia/official ban-list data, not an independent primary source.
- **The scraped site's own rules page**, `asia.pokemon-card.com/id/rules/regulation/`, under "Regulasi Luas": "Regulasi yang melingkupi kartu yang dijual sebelum Regulasi Standar..." ("The regulation covering cards sold before the Standard Regulation..."), listing the applicable product/card series as: Mega Evolution Series, Scarlet & Violet Series, Sword & Shield Series, and **Sun & Moon Series**. Notably it does **not** name Black & White or XY series. See the caveat in the Findings section below — this may mean the Indonesian/Asia card database itself doesn't catalog Black & White/XY-era cards at all, rather than a genuine rules difference.
- Effective date of the current D-floor: the 2021-23 season (Bulbapedia: "2021-23 — Cards with a regulation mark D or later, through Crown Zenith"). It has not moved in any subsequent season through 2026-27, confirmed by the identical "D or later" note carried through every later Standard-format history row (used there to describe Expanded's superset relationship) and by the Expanded-format page's own season-by-season list never changing.

## 4. Is there an official third format matching "Lainnya" ("Other")?

**No official third Regulation-Mark-defined format exists.** Play! Pokémon and the Japan/Asia regional sites recognize exactly **two** Regulation-Mark formats: Standard and Expanded. This was confirmed on three independent primary pages:

- Japan's official `pokemon-card.com/rules/regulation/` page (fetched live) has exactly two `<h2>` sections: "スタンダード" (Standard) and "エクストラ" (Extra = Expanded). No third Regulation-Mark format is documented there. (It separately documents a "殿堂レギュレーション" / "Hall of Fame Regulation" further down the page, but that is a **points-based restricted/banned-list format** for a specific promotional product line, not a Regulation-Mark range, and it isn't exposed as a search-filter option on the Asia site.)
- **The scraped site's own rules page**, `asia.pokemon-card.com/id/rules/regulation/`, likewise documents exactly two sections: "Regulasi Standar" and "Regulasi Luas." There is no "Regulasi Lainnya" heading or explanation anywhere on that page.
- Bulbapedia's own translation table for the *Standard format (TCG)* page gives the Indonesian localization string explicitly: `id=Standar`. The *Expanded format (TCG)* page's translation table gives `id=Luas`. Both match the site's radio-button labels exactly, confirming "Standar" = Standard and "Luas" = Expanded are the real official Indonesian format names, not the app's or my own guess. (https://bulbapedia.bulbagarden.net/wiki/Standard_format_(TCG) and https://bulbapedia.bulbagarden.net/wiki/Expanded_format_(TCG), "In other languages" section of each)
- Bulbapedia has **no** Indonesian localization entry at all on the *Unlimited format (TCG)* page — consistent with Unlimited not being an officially sanctioned Play! Pokémon tournament format in any region (it's recognized/permitted for casual League play but not run at sanctioned events), so it wouldn't have an official regional name to give "Lainnya."

**Candidates considered:**

- **Unlimited** — allows any card ever legal at any point (including pre-Regulation-Mark cards), with no mark restriction at all. Best conceptual fit for "everything not in Standard or Expanded," but it is not officially defined *by a Regulation Mark range* — it's defined by the *absence* of any Regulation Mark restriction. Bulbapedia, *Unlimited format (TCG)*: "allows any card that was once playable in Play! Pokémon tournaments... not used in any tournaments sanctioned by Play! Pokémon organized play." (https://bulbapedia.bulbagarden.net/wiki/Unlimited_format_(TCG))
- **Gym Leader Challenge (GLC)** — unofficial-but-Pokémon-recognized singleton/one-type deckbuilding format. Ruled out as the source of "Lainnya": GLC's *card pool* is explicitly the Expanded pool ("mainly derived from the Expanded format... cards from Black & White onward are legal"), plus deckbuilding restrictions (one Pokémon type, singleton, bans Rule Box Pokémon). It is not a distinct Regulation-Mark range — it's a deckbuilding sub-format layered on top of Expanded's card pool. A card-legality filter keyed on Regulation Mark ranges would have no reason to expose GLC as a separate bucket. (https://bulbapedia.bulbagarden.net/wiki/Gym_Leader_Challenge_format_(TCG))

**Best-supported conclusion:** nothing official defines "Lainnya" as a named format. It is most plausibly a UI catch-all for "everything not in Standard or Expanded" (which would functionally approximate Unlimited/no-restriction, i.e., pre-D cards and non-tournament products), **but empirically it is not even that** — see the live-query finding below, which found it currently matches **zero** cards on the actual site. Treat "Lainnya" as an under-documented/likely-vestigial UI option rather than a real third Regulation-Mark bucket, until/unless official documentation says otherwise.

### Live-query finding (first-party, from the actual scraped site)

I queried `asia.pokemon-card.com/id/card-search/list/` directly (POST, empty keyword, `cardType=all`) with each `regulation` value, on 2026-09-27:

| `regulation` value | Label | Result count |
|---|---|---|
| `1` | Standar | 4,152 |
| `2` | Luas | 12,565 |
| `3` | Lainnya | **0** |
| `all` | Semua | 12,586 |

Two things worth flagging for the data model:

1. `Semua` (12,586) minus `Luas` (12,565) = **21** cards that exist in the database but fall outside "Luas." If "Lainnya" were simply "everything not in Standard or Expanded," it should return those 21 cards — instead it returns **0**. This means "Lainnya," as currently wired on the live site, is not a complement/catch-all bucket either; it appears to be unused or reserved, matching no card in the current database.
2. The gap between `Semua` and `Luas` is only 21 cards out of 12,586 — i.e., Expanded ("Luas") covers effectively the entire Asia-site card database already. Combined with the rules page listing only Sun & Moon / Sword & Shield / Scarlet & Violet / Mega Evolution series under "Regulasi Luas" (no Black & White or XY series named), this suggests the Asia regional database may simply not catalog pre-Sun&Moon-era cards at all, rather than the site defining Expanded more narrowly than the global game does. I could not confirm this distinction further within this research pass — flagging it as an open question for whoever fixes the data model, since it affects whether "the 21 leftover cards" are old pre-mark oddities, non-tournament promotional/box products, or something else.

**Practical recommendation for the bug fix:** don't model "Lainnya" as a third Regulation-Mark range parallel to Standard/Expanded. Given it currently returns zero results, the safest data-modeling choice is to treat it as an unmapped/unknown bucket (or ignore it entirely) rather than inventing a semantic meaning for it that isn't backed by any official source or by the site's own current behavior.

## 5. Regulation Mark letter history

**Confirmed: marks start at A, not D — but A/B/C are Asia-exclusive and were never used on English-language (TPCi) cards.** D is the first mark that appears on English-region cards, which is presumably why some sources treat D as "the beginning" — but that's an English-market artifact, not the true start of the numbering scheme.

Per Bulbapedia's *Standard format (TCG)* page (fetched via API, "Regulation marks" section — this is the single most authoritative consolidated source found):

> "Asian expansions... introduc[ed] regulation marks on cards from the beginning of the Sun & Moon Series. ... The letter starts with A in Collection Sun and Collection Moon, and moves to the next letter upon each rotation."

> "Regulation marks were later introduced on cards released by The Pokémon Company International in the Sword & Shield Series. The first set featuring regulation marks was Sword & Shield, whose regulation mark is D in order to match Asian releases."

> "The A, B, and C regulation marks can be viewed in the Play! Pokémon Access app, but they cannot be specifically searched for and are otherwise not recognized in TPCi [English-language] regions."

Chronology reconstructed from Bulbapedia's rotation-history table (each row names the season and the new floor letter that started it — the floor letter is the newest letter at that point, i.e. approximately when it was introduced):

| Mark | Introduced (season / approx. calendar year) | Notes |
|---|---|---|
| A | 2017, Collection Sun / Collection Moon (Asia only) | Never used on English cards |
| B | ~2018-19 season (Asia only) | Never used on English cards |
| C | ~2019-20 season (Asia only) | Never used on English cards |
| D | Feb 2020, Sword & Shield (first English-market mark) | Rotated out of Standard 2023 |
| E | introduced within the 2021-23 season | Rotated out of Standard 2024 |
| F | introduced within the 2023-24 season | Rotated out of Standard 2025 |
| G | introduced within the 2024-25 season (Scarlet & Violet era) | Rotated out of Standard 2026 (effective Mar/Apr 2026) |
| H | introduced within the 2025-26 season | Standard-legal as of 2026-27 season |
| I | introduced within the 2025-26 season | Standard-legal as of 2026-27 season |
| J | introduced 2026 | Standard-legal as of 2026-27 season |

Caveat: I could not find a single authoritative source pinning an *exact* first set name + release date for each of E, F, G, H, I, J in this pass (the per-letter "first appeared in set X" detail is scattered across secondary card-database sites like pkmncards.com rather than stated plainly on one page). The season-by-season floor progression above, however, comes directly from Bulbapedia's own rotation table and is internally consistent with the official 2026 rotation announcement (which explicitly names H, I, J as currently legal and G as freshly rotated out).

**Sanity check on the claim "A, B, C, ... H, I, J is a plausible full range":** Confirmed plausible, with one important correction — **the range is real and unbroken (no skipped letters)**, but **A, B, and C are Asia/Japan-only marks that were never printed on English-language cards**. An app that only scrapes/serves English-market data would legitimately never see A/B/C in the wild; an app scraping an **Asian** locale site (as this one does) should expect to encounter A/B/C on very old Sun & Moon-era Asian cards, since asia.pokemon-card.com is exactly the kind of regional site where those marks are real and used. No letter is reported as skipped anywhere in these sources — the sequence runs unbroken A→J.

## Sources consulted

- Bulbapedia, *Standard format (TCG)* — https://bulbapedia.bulbagarden.net/wiki/Standard_format_(TCG) (wikitext fetched via Bulbapedia's own MediaWiki API, `action=parse&page=Standard%20format%20(TCG)`)
- Bulbapedia, *Expanded format (TCG)* — https://bulbapedia.bulbagarden.net/wiki/Expanded_format_(TCG) (same API method)
- Bulbapedia, *Unlimited format (TCG)* — https://bulbapedia.bulbagarden.net/wiki/Unlimited_format_(TCG)
- Bulbapedia, *Legacy format (TCG)* — https://bulbapedia.bulbagarden.net/wiki/Legacy_format_(TCG)
- Bulbapedia, *Gym Leader Challenge format (TCG)* — https://bulbapedia.bulbagarden.net/wiki/Gym_Leader_Challenge_format_(TCG)
- Pokemon.com, *2026 Pokémon TCG Standard Format Rotation Announcement* — https://www.pokemon.com/us/pokemon-news/2026-pokemon-tcg-standard-format-rotation-announcement
- Pokemon-card.com (Japan, official), *レギュレーション* (Regulation rules page) — https://www.pokemon-card.com/rules/regulation/ (fetched live)
- **asia.pokemon-card.com (the exact site the app scrapes), Indonesian locale**:
  - Rules/regulation explainer — https://asia.pokemon-card.com/id/rules/regulation/ (fetched live)
  - Card search form — https://asia.pokemon-card.com/id/card-search/ (fetched live, radio button markup)
  - Card search results endpoint, queried directly with each `regulation` value — https://asia.pokemon-card.com/id/card-search/list/ (POST, fetched live 2026-09-27)
- Secondary/community cross-checks (used only to corroborate, never as sole source): thegamer.com, onlygreats.com, tcgprotectors.com, pkmncards.com (regulation-mark pages)

## Where this file lives

Saved at `.scratch/cardstack-mvp/research/pokemon-tcg-regulation-marks.md`, as a sibling of the existing `issues/` and `plans/` directories in the same `cardstack-mvp` scratch folder, per the requested convention.
