# Series image source: pokepedia.id vs Bulbapedia

Research note, 2026-10-05. Question: which source gives a reliable, locale-appropriate image (logo) for each of the four Pokemon TCG Series in the Indonesian-scoped catalog (Scarlet & Violet, Sword & Shield "Pedang & Perisai", Sun & Moon "Matahari & Bulan", Mega Evolution "Evolusi Mega")? Context: `issues/15-series-image-source-research.md` and the regional-scoping invariant in `GLOSSARY.md` (Series entry). Everything below was observed live with `curl`/`jq` on 2026-10-05 unless marked "unverified". Image dimensions come from the MediaWiki `imageinfo` API and were re-checked with `file` on the downloaded bytes; I also opened the PNGs to confirm what each one shows.

## Summary and recommendation

**Pick: Bulbagarden Archives (the image repository behind Bulbapedia) for Scarlet & Violet, Sword & Shield and Mega Evolution, using the Indonesian-language Series logos. Sun & Moon has no Indonesian Series logo anywhere on Bulbapedia, so it needs a separate decision (section 5).** pokepedia.id could not be verified at all: it is not a MediaWiki, every page and endpoint I requested returned a Vercel bot-challenge (HTTP 429), so I have no evidence it has any Series image.

**Premise corrections.**

1. pokepedia.id is not a MediaWiki and has no `api.php`/`allimages`/`Special:FilePath`. It is a commercial Next.js card database plus marketplace operated by "PT Pasar Monster Saku" (section 2). It also sits behind a bot checkpoint.
2. Bulbapedia's own host (`bulbapedia.bulbagarden.net`) is behind a Cloudflare challenge for scripts (403 `cf-mitigated: challenge`), including its `api.php`. The images, and a working unauthenticated MediaWiki API, live on the sister site `archives.bulbagarden.net`, which is where Bulbapedia pulls its files from. So "query the Bulbapedia API" means "query the Archives API" in practice.
3. The messy-filename concern is real but narrow and filterable. For Series logos the Indonesian asset is cleanly named in 3 of 3 cases; the Thai and set-name mixing exists as separate files you must exclude (section 3).

**Top risks.** (a) Sun & Moon has no Indonesian Series logo. (b) Licence: these are contributor-claimed fair use of The Pokemon Company artwork, not an open licence (section 4). (c) Scarlet & Violet and Sword & Shield "Era" logos are small (398x249, 314x254) and carry a "Pokemon / Game Kartu Koleksi" lockup, unlike the Mega Evolution and Sword & Shield wordmarks. (d) Files can be renamed by Bulbapedia editors, which changes the URL (section 4).

## 1. Per-series results

| Series (Indonesian label) | Source | File title | Direct URL | Verified (HTTP, type, size) | Region of the asset |
|---|---|---|---|---|---|
| Scarlet & Violet | Archives | `File:SV Era Logo I.png` (page text: "Scarlet & Violet Series logo in Indonesian") | https://archives.bulbagarden.net/media/upload/3/3b/SV_Era_Logo_I.png | 200, `image/png`, 398x249, 91,981 bytes | Indonesian (text "GAME KARTU KOLEKSI" under the Pokemon logo, then the Scarlet & Violet wordmark, which is the same in Indonesian) |
| Sword & Shield (Pedang & Perisai) | Archives | `File:Sword Shield Logo Indonesian.png` (preferred, "Indonesian Sword & Shield logo") | https://archives.bulbagarden.net/media/upload/6/6c/Sword_Shield_Logo_Indonesian.png | 200, `image/png`, 1500x545, 558,060 bytes | Indonesian ("PEDANG & PERISAI" wordmark with the V) |
| Sword & Shield (alternative) | Archives | `File:SS Era Logo I.png` ("Sword & Shield Series logo in Indonesian") | https://archives.bulbagarden.net/media/upload/3/3b/SS_Era_Logo_I.png | 200, `image/png`, 314x254, 335,432 bytes | Indonesian (Pokemon / Game Kartu Koleksi lockup plus "PEDANG & PERISAI") |
| Sun & Moon (Matahari & Bulan) | Archives | none for Indonesian. Only `File:SM Era Logo J.png` (Japanese), `File:SM Era Logo K.png` (Korean), `File:SM Series Logo SC.png` (Simplified Chinese) | https://archives.bulbagarden.net/media/upload/2/20/SM_Era_Logo_J.png (Japanese, not usable) | 200, `image/png`, 540x137 (the Japanese one) | Japanese / Korean / Chinese only. There is no Indonesian or Thai Series logo. Indonesian Sun & Moon only exists as per-set logos, e.g. `File:First Impact Logo Indonesian.png` |
| Mega Evolution (Evolusi Mega) | Archives | `File:Mega Evolution Logo Indonesian.png` ("Mega Evolution (Evolusi Mega) Indonesian logo") | https://archives.bulbagarden.net/media/upload/d/d3/Mega_Evolution_Logo_Indonesian.png | 200, `image/png`, 880x320, 47,309 bytes | Indonesian ("EVOLUSI MEGA" wordmark, no Thai script) |
| All four | pokepedia.id | not determinable | cdn2.pokepedia.id is referenced in the page head, but no image path was observable | every page returned 429 `x-vercel-mitigated: challenge` | unverified |

The Mega Evolution Thai trap is `File:MA1 Mega Evolution Logo Indonesian Thai.png` (880x700, https://archives.bulbagarden.net/media/upload/6/6d/MA1_Mega_Evolution_Logo_Indonesian_Thai.png). Its description reads "Mega Evolution Indonesian/Thai logo" and the image stacks the Indonesian wordmark above Thai script. Do not use it.

Source for the per-file facts: the Archives API, e.g. `https://archives.bulbagarden.net/w/api.php?action=query&titles=File:SV%20Era%20Logo%20I.png&prop=imageinfo|revisions&iiprop=url|size|mime&rvprop=content&rvslots=main&format=json`. The per-language "Era Logo" suffix is not uniform: `SV Era Logo F` is Traditional Chinese, `SS Era Logo C` is Simplified Chinese, `SS Era Logo F` is Traditional Chinese, `I` = Indonesian, `T` = Thai, `J` = Japanese, `K` = Korean (read from each file's description). So never infer the language from the letter; read the description or the title word "Indonesian".

## 2. pokepedia.id

Findings, in order of how decisive they are.

- **Not a MediaWiki.** The site title is "pokepedia.id Ensiklopedia Kartu Pokemon Terlengkap". The nav is Beranda, Pencarian, Market, Portofolio, Jual; the footer reads "(c) 2026 PT Pasar Monster Saku" and links `/terms/syarat-dan-ketentuan` and `/terms/kebijakan-privasi`. It is a Next.js app (`/_next/static/...`), and the paths seen are `/expansions`, `/market`, `/portfolio/collection`, `/scan`, `/seller`, `/login`. Source: the unchallenged 404 body at https://www.pokepedia.id/sitemap.xml (fetched with curl, then read as HTML). `https://pokepedia.id/` redirects (308) to `https://www.pokepedia.id/`.
- **Bot checkpoint on everything else.** `curl` and WebFetch against `/`, `/api.php`, `/expansions`, `/expansions/ma3`, `/terms/syarat-dan-ketentuan` all returned HTTP 429 with `x-vercel-mitigated: challenge` and the title "Vercel Security Checkpoint" (WebFetch got 429/403). I did not try to get past the checkpoint, for example by driving a real browser: it is bot detection, and getting past it is not something this research should do or a production ingester could rely on.
- **robots.txt** (https://www.pokepedia.id/robots.txt, 200): `User-Agent: *`, `Allow: /`, `Disallow: /api/`, `/portfolio/`, `/settings`; and a separate block that disallows the whole site (`Disallow: /`) for GPTBot, ChatGPT-User, ClaudeBot, Claude-Web, CCBot, PerplexityBot, Bytespider, AhrefsBot, SemrushBot, and several others. A scraper for an AI-assisted pipeline is explicitly unwelcome by that file.
- **Granularity.** Web search results (not primary) show pages like `https://www.pokepedia.id/expansions/ma3` ("Evolusi Mega Impian ex"), `/expansions/ma1` ("Evolusi Mega"), `/expansions/sc1b` ("Pedang & Perisai B") and `/expansions/s-p` ("Kartu Promo Pedang & Perisai"). These are per-Expansion-Set pages. I found no Series page and no Series logo; whether `/expansions` groups sets under Series with a logo is unverified.
- **Images** would come from `https://cdn2.pokepedia.id` (a `preconnect`/`dns-prefetch` link in the page head; the bare host returns a 404 page, `cache-control: public, max-age=86400`). No concrete image URL was observed. Stability, licence and hotlink policy: unverified, and the Terms page could not be read (403/429).
- **Licence / ToS:** unverified, the Terms page is behind the checkpoint. It is a commercial marketplace, so assume all rights reserved until the Terms say otherwise.

Reproducible approach for pokepedia.id: none from a script. A maintainer would have to read `/expansions` in a normal browser and, if a Series logo exists, copy its `cdn2.pokepedia.id` URL by hand, then either self-host it or ask permission via `admin@pokepedia.id` (the contact listed on the site). I would not build an ingester on it.

## 3. Bulbapedia (Bulbagarden Archives): messy filenames and a cleaning rule

How the mess looks (real titles from `list=search&srnamespace=6` and `list=allimages` on the Archives API, 2026-10-05):

- **Series logos use two schemes.** `SV Era Logo I.png`, `SS Era Logo I.png`, `SM Era Logo J.png` ("<code> Era Logo <letter>"), and `Sword Shield Logo Indonesian.png`, `Mega Evolution Logo Indonesian.png` ("<Series> Logo <Language word>"). The first scheme uses a one-letter language code; the second spells the language out.
- **Set names are mixed into the same namespace and category.** The category `Category:TCG Scarlet & Violet set logos` (30 files) holds Series logos and per-set logos together: `Scarlet ex Logo Indonesian.png`, `Violet ex Logo Indonesian.png`, `Stellar Guidance Logo Indonesian.png`, `Black Shine Logo Indonesian.png`, `Pokémon Card 151 Logo Indonesian.png`, and so on. `Category:TCG Sword & Shield set logos` has 71 files, mixing `Star Birth Logo Indonesian.png`, `Fusion Arts Logo Indonesian.png`, `SWSH1 Logo BR.png` (Brazilian), `Champion Path Logo BR.png`, with the Series logos. `Category:TCG Mega Evolution set logos` has only 5 files: `Mega Evolution Logo Indonesian.png`, `MA1 Mega Evolution Logo Indonesian Thai.png`, `MEGA Dream ex Logo Indonesian.png`, `Blue Blaze Logo Indonesian.png`, `Void Blast Logo Indonesian.png`.
- **Thai mixed in, two ways.** Separate Thai files: `SV Era Logo T.png`, `SS Era Logo T.png`, `Sword Shield Logo Thai.png`, and per-set Thai logos like `First Impact Logo Thai.png`. And one file that combines both: `MA1 Mega Evolution Logo Indonesian Thai.png` (Indonesian plus Thai script in one image).
- **Category membership is incomplete.** `SV Era Logo J.png` is not in `TCG Scarlet & Violet set logos` (it lives in `Category:Japanese Scarlet & Violet Series logos`), and `SM Series Logo SC.png` is in `Category:Simplified Chinese Sun & Moon Series logos`. So filter by file title, not only by category.
- **Language codes are inconsistent** (`C` vs `F` for the two Chinese scripts, see section 1).

Cleaning rule. Do not scan a whole category and guess. Use a fixed allowlist of four titles (for these four Series, which do not change) and verify it with a query. The query below discovers all Series-scheme candidates, and the allowlist then picks the Indonesian one:

```sh
A=https://archives.bulbagarden.net/w/api.php
UA='cardstack-ingester/1.0 (contact: <maintainer email>)'
for c in "Scarlet & Violet" "Sword & Shield" "Sun & Moon" "Mega Evolution"; do
  curl -sS -A "$UA" -G "$A" --data-urlencode action=query --data-urlencode generator=categorymembers \
    --data-urlencode "gcmtitle=Category:TCG $c set logos" --data-urlencode gcmtype=file --data-urlencode gcmlimit=500 \
    --data-urlencode prop=imageinfo --data-urlencode iiprop='url|size|mime' --data-urlencode format=json |
  jq -r --arg s "$c" '.query.pages[]
    | select(.title | test("^File:(SV|SS|SM|ME) Era Logo [A-Z]\\.png$|^File:(Scarlet Violet|Sword Shield|Sun Moon|Mega Evolution) Logo [A-Za-z]+\\.png$"))
    | [$s, .title, "\(.imageinfo[0].width)x\(.imageinfo[0].height)", .imageinfo[0].url] | @tsv'
done
```

I ran this exact command; it prints 16 rows (4 Scarlet & Violet, 9 Sword & Shield, 2 Sun & Moon, 1 Mega Evolution). The regexes keep Series-scheme files and drop all set-named files (`Stellar Guidance Logo Indonesian.png` etc.) and the set-code ones (`SV1 Logo EN.png`, `MA1 ...`). To pick the Indonesian one, then take either the one-letter code `I` or the word `Indonesian`, and exclude any title containing `Thai`:

```sh
... | grep -E $'\tFile:(S[VSM] Era Logo I|[A-Za-z ]+ Logo Indonesian)\\.png\t' | grep -v Thai
```

That yields, in this order: `SV Era Logo I.png` (Scarlet & Violet), `SS Era Logo I.png` and `Sword Shield Logo Indonesian.png` (Sword & Shield, two valid files, take the 1500x545 wordmark for a heading), `Mega Evolution Logo Indonesian.png`. It correctly returns nothing for Sun & Moon. It also drops `MA1 Mega Evolution Logo Indonesian Thai.png`: the first command's regex rejects it (the title starts with `MA1`, not `Mega Evolution`), and the `grep -v Thai` is a second guard. I ran this pipeline and it returned exactly those four rows.

Because the Series list is fixed and tiny, the simplest durable form is a hard-coded map from Series name to the four direct URLs above, re-verified by that command, rather than a runtime API call at ingestion.

URL stability.

- **Hash paths, not redirects.** The direct URL is `https://archives.bulbagarden.net/media/upload/<h>/<hh>/<File_name>`, where `<h>` and `<hh>` are the first one and two hex characters of the MD5 of the underscored filename (verified: MD5 of `SV_Era_Logo_I.png` starts with `3b`, `Sword_Shield_Logo_Indonesian.png` with `6c`, `Mega_Evolution_Logo_Indonesian.png` with `d3`). A re-upload of the same title keeps the URL (the old file moves to `/media/upload/archive/...`, e.g. `SV_Era_Logo_I.png` has two revisions, 2023-06-03 and 2023-06-08, by user 4iamking). A rename (page move) changes the URL; renames are routine (Archives move log shows files moved on 2026-10-03 and 2026-10-05). So the URL is stable until someone renames the file or replaces the artwork.
- **`Special:FilePath` is not usable from a script.** `https://archives.bulbagarden.net/wiki/Special:FilePath/SV_Era_Logo_I.png` returned 403 with `cf-mitigated: challenge`. The `/media/upload/...` URLs and `/w/api.php` returned 200 to a plain `curl` with a descriptive User-Agent and with a Firefox User-Agent.
- **Response headers on the image** (all five direct URLs): `content-type: image/png`, `access-control-allow-origin: *`, `cache-control: public, max-age=2592000, s-maxage=3600`. A request with a foreign `Referer` (`https://cardstack.example/`) still returned 200 `image/png`, so there is no referer-based hotlink block today.
- **robots.txt:** `https://archives.bulbagarden.net/robots.txt` itself returned a Cloudflare challenge (403), so I could not read it. Unverified. I found no hotlink or API-usage policy page on the Archives wiki (search for "hotlink" in the main, project and template namespaces returned nothing).

## 4. Licence and production use

- **Bulbagarden Archives** (`Archives:Copyrights`, read through the API): "The preferred license is the Creative Commons Attribution-NonCommercial-ShareAlike license; however, most items are not licensed at all and are claimed as fair use." Every file checked here carries `{{i-Fairuse-tcg-noncard}}`, rendered as: "This image is from the Pokemon TCG, or substantially derived from it. The contributor claims this to be fair use." So these logos are Pokemon Company artwork under a contributor's fair-use claim; there is no CC BY-NC-SA grant on these files, and CC BY-NC-SA would not fit a production app with any commercial angle anyway. Disclaimer text: "Pokemon (c) 2002-... Pokemon. (c) 1995-... Nintendo/Creatures Inc./GAME FREAK inc. ... No copyright or trademark infringement is intended". The site's `rightsinfo` is empty. Sources: https://archives.bulbagarden.net/wiki/Archives:Copyrights, https://archives.bulbagarden.net/wiki/Template:I-Fairuse-tcg-noncard.
- **Practical reading.** The underlying rights holder is the same one whose card images pokemonasia (the ingestion source) already serves and this app already shows, so using the wordmark is no new kind of exposure; but "fair use" on a wiki is not a licence for this app. If the maintainer wants zero ambiguity, an official Pokemon Indonesia / AKG Entertainment press kit is the clean source (not researched).
- **Hotlink vs self-host.** Hotlinking works today (headers above), but it puts bandwidth on a volunteer wiki, depends on filenames nobody here controls, and there is no policy document I could read. Four small PNGs (47 to 558 KB) are easy to self-host in `frontend/public/` and then the URL stays stable, which also avoids a Cloudflare challenge ever blocking the image in the browser. That is a maintainer call; I lean self-host or at minimum a stored absolute URL plus an empty-string fallback.
- **pokepedia.id:** licence and Terms unverified (section 2), robots.txt disallows AI crawlers.

## 5. The Sun & Moon gap

Bulbapedia has no Indonesian Sun & Moon Series logo (checked: `Category:TCG Sun & Moon set logos` has 64 files, only `SM Era Logo J` and `SM Era Logo K` at Series level, plus per-set Indonesian and Thai logos such as `First Impact Logo Indonesian.png`, `Hidden Shadow Logo Indonesian.png`, `Legends Awakened Logo Indonesian.png`, `Sky Ruler Logo Indonesian.png`). TCGdex's Indonesian locale has no Sun & Moon or Mega Evolution Series at all and no `logo` on the two it has (re-queried today: `api.tcgdex.net/v2/id/series` returns only `SV` "Scarlet & Violet" and `S` "Pedang & Perisai", both `logo: null`), as already recorded in ticket 15. Options, cheapest first:

1. Leave Sun & Moon with an empty `ImageURL` and render the Series heading as text only (the same graceful-empty behaviour #14 specified). Costs nothing, the gap is one of four.
2. Use the TCGdex English/international Sun & Moon logo as a stated proxy for that one Series (the ticket's earlier finding), accepting the locale mismatch for one row only.
3. Ask the maintainer to look at pokepedia.id's `/expansions` by hand, or contact the Indonesian licensee, for an Indonesian Sun & Moon wordmark.
4. Crop or use the Indonesian per-set logos. Not recommended: they show set names, not the Series.

I would take option 1 for now, since the schema field is empty-string by default anyway.

## 6. What a `ready-for-agent` follow-up ticket would need

Mirror ticket #14 (`ExpansionSet.ImageURL`), with one structural difference: Series logos do not come from the pokemonasia listing, so there is nothing to scrape.

1. `Series` gains `ImageURL string`, `not null default ''`, same convention as `ExpansionSet.ImageURL`/`Card.ImageURL`; `SeriesSummary` exposes `imageUrl`; the catalog browse page's per-Series heading shows it when non-empty, no broken-image icon and no layout shift when empty.
2. Population: a fixed map in the backend (ingestion or a migration seed) from the Series name as pokemonasia labels it ("Scarlet & Violet", "Pedang & Perisai", "Matahari & Bulan", "Evolusi Mega" - confirm the exact strings against the DB) to the URL; no HTTP call at ingestion time. A Series with no entry keeps `""`. The Series upsert path writes it on insert and update so a full ingestion backfills it, like #14.
3. Decide hosted vs hotlinked before building: either store the four Archives URLs from section 1 as-is, or download the four files into `frontend/public/` and store site-relative paths (then there is no external dependency). Either way, a short code comment or ADR line should say "Indonesian logo from Bulbagarden Archives, contributor-claimed fair use" so the provenance is not lost.
4. Acceptance: for each of the four Series the stored value is the expected URL or empty; the three Indonesian URLs return 200 `image/png` (a one-off check or a test that fetches them is optional); Sun & Moon renders text-only if option 1 is chosen.
5. A small re-verification note: the three Archives titles can be renamed, so keep the discovery command from section 3 in the ticket or a doc and re-run it if an image goes missing.

## 7. Evidence gaps

- pokepedia.id: no page, API, image or Terms content was readable, every request got a Vercel challenge (429) or 403, so it is unverified whether a Series image exists there. The conclusion "not a MediaWiki" comes from the site's own HTML (404 page) and its robots.txt, not from the live expansions pages.
- `archives.bulbagarden.net/robots.txt` and any hotlink policy: unreadable (Cloudflare challenge). Bulbapedia's own `api.php` is also challenged; only the Archives host answers scripted requests.
- I did not test sustained or rate-limited access; the API returned no rate-limit headers in the checks above.
- Licence analysis is a reading of the wiki's own copyright pages, not legal advice.

## Addendum: Sun & Moon via First Impact logo (maintainer-supplied)

The maintainer found an Indonesian Sun & Moon logo: [`File:First_Impact_Logo_Indonesian.png`](https://bulbapedia.bulbagarden.net/wiki/File:First_Impact_Logo_Indonesian.png) (set page: [First Impact (ATCG)](https://bulbapedia.bulbagarden.net/wiki/First_Impact_(ATCG))), direct URL `https://archives.bulbagarden.net/media/upload/7/71/First_Impact_Logo_Indonesian.png`.

Verified by fetch: HTTP 200, `image/png`, 1177x298 RGBA (transparent background), about 466 KB.

It is a set logo, not a Series logo: the top wordmark reads "MATAHARI&BULAN" (the Series name) and the bottom one reads "Hantaman Pertama" (the set name). The two are separated by a fully transparent band at rows 182 to 188, so a crop to rows 0-182 (1177x182) isolates the Series wordmark cleanly.

Consequence: this one needs a derived, self-hosted asset (cropped copy), not a hotlink. That inherits the fair-use licence caveat above and makes the follow-up ticket's asset handling two-track: hotlink or self-host the three Series logos as-is, plus one cropped file for `sm`. Alternatively self-host all four for uniformity.
