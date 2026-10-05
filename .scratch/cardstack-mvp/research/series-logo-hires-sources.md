# Higher-resolution Indonesian Series logo sources

Research note, 2026-10-05, follow-up to `series-image-source.md` (same folder). Question: is there a higher-resolution, Indonesian-language, lockup-free source for the Scarlet & Violet, Sword & Shield ("Pedang & Perisai") and Mega Evolution ("Evolusi Mega") Series wordmarks, using the Matahari & Bulan approach (crop the Series wordmark out of an Indonesian per-set logo) or any other source? Everything below was observed live on 2026-10-05 with `curl` (descriptive User-Agent) and Python PIL/`vips`, unless marked "unverified". Every candidate listed was downloaded and opened (Read) to confirm what it shows; alpha bands come from PIL (alpha > 16, threshold 0 and 128 give the same rows within 1 px).

## Summary and recommendation

**Short answer: there is no better source for any of the three. The current source files are already the largest Indonesian Series wordmarks that exist on Archives or on the official Indonesian site.** Only Scarlet & Violet has a real problem, and that is fixable by cropping the existing file.

1. **Scarlet & Violet: keep `SV_Era_Logo_I.png`, crop out the lockup with `vips crop ... 0 193 398 56`.** The wordmark is a clean 398x56 band (rows 193-249) below a 23 px fully transparent gap (rows 170-193) under the "Game Kartu Koleksi" bar. No Indonesian per-set logo contains the "Scarlet & Violet" wordmark (Scarlet ex / Violet ex show only the set name), so the Matahari & Bulan approach does not apply here. 398x56 is the native ceiling: at the current 80 px delivery height it becomes 569x80, an upscale of 1.43x (looks slightly soft but fine; compared by eye against the current 128x80 lockup webp). A 737x432 older revision of the same file exists on Archives (about 1.85x linear) but its URL is Cloudflare-challenged for scripts, so it is unverified and not scriptable (section 2).
2. **Sword & Shield (Pedang & Perisai): no change needed.** `Sword_Shield_Logo_Indonesian.png` is 1500x545, already lockup-free, tight to the alpha edge (rows 0-545, cols 0-1500). The current 220x80 webp is not low-resolution; it is 80 px tall because the build script delivers every logo at 80 px (the heading renders at `h-10`, 40 CSS px, so 80 px is exactly 2x). The source supports up to 545 px height.
3. **Mega Evolution (Evolusi Mega): no change needed, optional tight trim.** `Mega_Evolution_Logo_Indonesian.png` is 880x320 with 5 px transparent margin top and bottom; the trimmed wordmark is 867x308 (crop box 6,5,867,308 gives 225x80 instead of 220x80, a cosmetic 2 percent gain). The only other Indonesian Mega candidate with the wordmark, `MEGA Dream ex Logo Indonesian.png`, stacks "EVOLUSI MEGA" (about 93 px tall) over "IMPIAN ex" with overlapping outlines and no transparent gap, so it is both smaller and not croppable.
4. **No first-party source helps.** The Indonesian official site (`asia.pokemon-card.com/id`) shows Series only as text; it exposes pack/product photos, hero art and campaign banners, not Series wordmarks (section 4).
5. **Resolution headroom if the delivery height is ever raised.** Maximum clean delivery height per Series: Scarlet & Violet 56 px (after crop), Matahari & Bulan 182 px, Evolusi Mega 308 px, Pedang & Perisai 545 px. At the current 80 px (2x of 40 CSS px) all four are at or above native resolution except Scarlet & Violet, which is upscaled 1.43x.

## 1. Delivery constraint (why "small" is relative)

`backend/scripts/build-series-logos.sh` runs `vips thumbnail ... 100000 --height 80` on each source, so every asset is 80 px tall and only the width varies. `frontend/src/components/catalog/series-heading.tsx` renders the logo with `height={40}` and `className="h-10 w-auto max-w-full"`. Current assets, measured with PIL:

| Asset | Pixel size | Bytes | Source | Source pixel size | Scale applied |
|---|---|---|---|---|---|
| `scarlet-violet.webp` | 128x80 | 7,858 | `SV_Era_Logo_I.png` whole file (lockup included) | 398x249 | 0.32x (the wordmark itself ends up about 128x18 px) |
| `pedang-perisai.webp` | 220x80 | 8,652 | `Sword_Shield_Logo_Indonesian.png` | 1500x545 | 0.147x |
| `evolusi-mega.webp` | 220x80 | 10,084 | `Mega_Evolution_Logo_Indonesian.png` | 880x320 | 0.25x |
| `matahari-bulan.webp` | 517x80 | 19,386 | `First_Impact_Logo_Indonesian.png` rows 0-182 | 1177x182 | 0.44x |

So Pedang & Perisai and Evolusi Mega are downscaled, not upscaled; they only look small because of the 80 px height. The Scarlet & Violet asset is the actual problem: the lockup consumes 78 percent of the image height, leaving the wordmark about 18 px tall in the webp.

## 2. Scarlet & Violet

### Candidates

All Indonesian files in `Category:TCG Scarlet & Violet set logos` (API: `action=query&generator=categorymembers&gcmtitle=Category:TCG Scarlet & Violet set logos&gcmtype=file&gcmlimit=500&prop=imageinfo&iiprop=url|size|mime`, 30 members; Thai-suffixed ones skipped). All downloaded, all HTTP 200 `image/png`. None of the per-set logos carry the Series wordmark: they show only the set name (checked by opening each). Licence for all: `{{i-Fairuse-tcg-noncard}}` (read via `prop=revisions&rvprop=content&rvslots=main` for `SV Era Logo I`, `Scarlet ex Logo Indonesian`, `SV1 Logo EN`; the others inherit the same template per the earlier note).

| Title | Direct URL | Size | Bytes | Contains the "Scarlet & Violet" wordmark | Crop box (x,y,w,h) | Licence |
|---|---|---|---|---|---|---|
| `SV Era Logo I.png` | https://archives.bulbagarden.net/media/upload/3/3b/SV_Era_Logo_I.png | 398x249 | 91,981 | Yes. Three alpha bands: rows 0-122 Pokemon logo, rows 126-170 "Game Kartu Koleksi" bar, rows 193-249 the "Scarlet & Violet" wordmark (cols 0-398, outline touches both side edges: alpha 213 and 242 on the edge columns, so the art is slightly clipped at the left and right) | **0,193,398,56** | contributor-claimed fair use |
| `SV Era Logo I.png` older revision 2023-06-03 (`archivename` `20230608074251!SV_Era_Logo_I.png`) | https://archives.bulbagarden.net/media/upload/archive/3/3b/20230608074251%21SV_Era_Logo_I.png | 737x432 (per API) | 1,293,117 (per API) | Presumably the same artwork with the lockup, opaque ("Bad image: Needs Transparency" in its upload comment); unverified | would need a background removal plus a crop of about rows 335-432 (estimate, about 104 px tall wordmark) | same page |
| `Scarlet ex Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/9/97/Scarlet_ex_Logo_Indonesian.png | 891x261 | 249,824 | No: "SCARLET ex" set name only | n/a | fair use |
| `Violet ex Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/3/38/Violet_ex_Logo_Indonesian.png | 835x265 | 210,299 | No: "VIOLET ex" | n/a | fair use |
| `Triplet Beat`, `Snow Hazard`, `Clay Burst`, `Ace Paradox`, `Paradox Encounters`, `Transfiguration Mask`, `Bonds of Destiny`, `Stellar Guidance`, `Stellar Lightning`, `Black Shine`, `Shiny Treasure ex`, `Terastal Fest ex`, `Pokémon Card 151` (all `... Logo Indonesian.png`), `SV5s Ace Paradox`, `SV6s Transfiguration Mask`, `SV11s Black White` | see `list=categorymembers` call above | 450x100 to 2400x1700 | 16 KB to 1.98 MB | No: each shows only the Indonesian set name (e.g. "BIMBINGAN RASI", "KILAU HITAM", "HITAM & PUTIH") | n/a | fair use |
| `SV1 Logo EN.png` (English, last resort only) | https://archives.bulbagarden.net/media/upload/7/72/SV1_Logo_EN.png | 898x260 | 220,606 | The text "SCARLET & VIOLET" in the English set-logo style, on a crystal backdrop that cannot be separated from the letters. It is the base set logo, not the Series logo. The text is language-neutral | none (alpha rows 1-259, cols 0-896) | fair use, "Designed by Huy Cao", source tcg.pokemon.com `sv01-logo-2x.png` per the upload comment |

Other-language Series logos (`SV Era Logo J/F/K/T`: 549x138, 916x709, 183x122, 356x240) are not acceptable: J, F and K carry Japanese, Chinese and Korean script; T carries Thai script plus the Pokemon lockup (opened all four).

Results from the three searches that could have found a hidden Indonesian Series file: `list=search&srnamespace=6&srsearch=Scarlet & Violet logo` (50 hits, only `SV Era Logo I` is Indonesian Series-level), `Category:English Scarlet & Violet Series logos` (18 files, all set logos), `Category:Japanese Scarlet & Violet Series logos` (24 files), `Category:TCG logos` (8 files: Pokemon TCG lockups for Indonesia/Thailand/Korea/Greater China, nothing Series-level). `list=allcategories&acprefix=Indonesian Scarlet` returns only `Indonesian Scarlet & Violet Series set symbols` (no logo category).

### Fallback result: crop of `SV Era Logo I.png`

```sh
vips crop SV_Era_Logo_I.png sv-wordmark.png 0 193 398 56
vips thumbnail sv-wordmark.png scarlet-violet.webp[Q=90] 100000 --height 80
```

I ran exactly this: crop is 398x56 RGBA with a fully populated alpha (rows 0-56, cols 0-398, so no leftover transparent margin and nothing cut), and the 80 px webp is 569x80, 14,502 bytes (vs 128x80, 7,858 bytes today). A 40 px version is 284x40, 7,432 bytes. In the script this would be added next to the Sun & Moon crop (the same `vips crop` + `mv` pattern).

Quality caveat: 56 px native height means the 80 px delivery is a 1.43x upscale and the edges are a little soft; 1x (40 CSS px, 56 px native) is crisp. If higher fidelity is wanted, the options are (a) accept; (b) have the maintainer download the 737x432 older revision in a normal browser (scripted fetch of any `/media/upload/archive/...` URL returns the Cloudflare challenge, HTTP 403 `cf-mitigated: challenge`, tried both `%21` and `!` spellings and the thumb path) and remove its background by hand, which would give about 104 px; (c) use `SV1 Logo EN.png` (258 px tall) as a language-neutral stand-in, but it is a different visual style from the other three Indonesian logos (crystal set logo, not the era wordmark), so I do not recommend it.

## 3. Sword & Shield (Pedang & Perisai)

All Indonesian files in `Category:TCG Sword & Shield set logos` were downloaded (23 files, all HTTP 200 `image/png`). The category call is the same as above with `gcmtitle=Category:TCG Sword & Shield set logos` (71 members).

| Title | Direct URL | Size | Bytes | Contains the "Pedang & Perisai" wordmark | Crop box (x,y,w,h) | Licence |
|---|---|---|---|---|---|---|
| `Sword Shield Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/6/6c/Sword_Shield_Logo_Indonesian.png | 1500x545 | 558,060 | Yes, it is the whole image: "PEDANG & PERISAI" with the large V; single alpha band rows 0-545, cols 0-1500; no Pokemon TCG lockup | none (already tight) | fair use (`{{i-Fairuse-tcg-noncard}}`, page says "Indonesian Sword & Shield logo"); one revision only (uploaded 2020-12-12 by Nuva-kal) |
| `SS Era Logo I.png` | https://archives.bulbagarden.net/media/upload/3/3b/SS_Era_Logo_I.png | 314x254 | 335,432 | Yes, but with the lockup: alpha bands rows 1-114 Pokemon logo, rows 117-159 "Game Kartu Koleksi" bar, rows 172-253 "PEDANG & PERISAI" (cols 65-262, so only 197x81) | 65,172,197,81 | fair use (its upload comment misnames it "Sun & Moon Series logo in Indonesian") |
| `Shiny VMAX Collection Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/0/02/Shiny_VMAX_Collection_Logo_Indonesian.png | 418x148 | 94,618 | A tiny "PEDANG&PERISAI" line (about 20 px tall) over the set name "KOLEKSI VMAX BERKILAU"; one alpha band, so not separable | n/a | fair use |
| `Star Birth`, `Fusion Arts`, `VSTAR Universe`, `Eevee Heroes`, `Battle Region`, `Peerless Fighters`, `Blue Sky Stream`, `Dark Phantasma`, `Incandescent Arcana`, `Jet-Black Spirit`, `Lost Abyss`, `Paradigm Trigger`, `Rapid Strike Master`, `Single Strike Master`, `Silver Lance`, `Skyscraping Perfection`, `Space Juggler`, `Time Gazer`, `VMAX Climax`, `25th Anniversary Collection` (all `... Logo Indonesian.png`) | see category call | 320x80 to 840x240 | 29 KB to 356 KB | No: set names only | n/a | fair use |

`Sword Shield Logo Thai.png` (828x321) and the Chinese file are excluded (Thai or Chinese script). I found no larger Indonesian version: `prop=imageinfo&iilimit=max` on `File:Sword Shield Logo Indonesian.png` returns one revision.

Conclusion: the current source is the best and needs no crop. Optional only: deliver more pixels (for example `--height 160` gives 440x160, 23,030 bytes, measured) if the heading ever grows.

## 4. Mega Evolution (Evolusi Mega)

`Category:TCG Mega Evolution set logos` has 5 members (4 downloaded and opened, HTTP 200 `image/png`; the Thai combination file was excluded by title and only checked in the earlier note).

| Title | Direct URL | Size | Bytes | Contains the "Evolusi Mega" wordmark | Crop box (x,y,w,h) | Licence |
|---|---|---|---|---|---|---|
| `Mega Evolution Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/d/d3/Mega_Evolution_Logo_Indonesian.png | 880x320 | 47,309 | Yes, it is the whole image (stacked "EVOLUSI / MEGA", yellow outline); single alpha band rows 5-313, cols 6-873 | optional trim **6,5,867,308** | fair use (`{{ATCG\|Mega Evolution}} (Evolusi Mega) Indonesian logo`); one revision (2025-08-21, ShiroKuro-chan) |
| `MEGA Dream ex Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/1/10/MEGA_Dream_ex_Logo_Indonesian.png | 840x280 | 322,253 | Yes as the top line "EVOLUSI MEGA" (about 93 px tall, 826 px wide), over "IMPIAN ex". Alpha is one continuous band (rows 2-273) and the row coverage never drops to zero between the lines (outlines overlap), so a clean crop is not possible, and it is smaller than the standalone file anyway | none usable | fair use |
| `Blue Blaze Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/a/a0/Blue_Blaze_Logo_Indonesian.png | 810x210 | 213,342 | No: "KOBARAN BIRU" | n/a | fair use |
| `Void Blast Logo Indonesian.png` | https://archives.bulbagarden.net/media/upload/7/7b/Void_Blast_Logo_Indonesian.png | 892x230 | 43,457 | No: "LEDAKAN PENIADA" | n/a | fair use |
| `MA1 Mega Evolution Logo Indonesian Thai.png` | https://archives.bulbagarden.net/media/upload/6/6d/MA1_Mega_Evolution_Logo_Indonesian_Thai.png | 880x700 | 78,349 | Yes, but stacked above Thai script; excluded | n/a | fair use |

English `ME1 Logo EN.png` (1920x386, source press.pokemon.com per the file history) exists but reads "MEGA EVOLUTION"; "Evolusi Mega" is not language-neutral, so it is not an acceptable replacement.

Trim result (run): `vips crop Mega_Evolution_Logo_Indonesian.png me.png 6 5 867 308` then `--height 80` gives 225x80, 10,124 bytes (current 220x80, 10,084 bytes). Gain is cosmetic.

## 5. Sun & Moon (reference only)

`matahari-bulan.webp` is the existing good case: `First_Impact_Logo_Indonesian.png` (1177x298, 465,950 bytes) has alpha bands rows 0-182 (cols 0-1177) and rows 189-298 (cols 28-1148), so the crop 0,0,1177,182 is exact and full-bleed; the webp is 517x80. Native ceiling 182 px height. No other Sun & Moon Indonesian file is larger (the other per-set logos are 124x44 to 158x41).

## 6. First-party and other sources

Observed 2026-10-05, plain `curl` with a Firefox User-Agent unless noted.

- `https://asia.pokemon-card.com/id/` (200, `text/html`), `/id/card-search/` (200), `/id/products/` (200): Series appear only as text in the filter ("Scarlet & Violet", "Pedang & Perisai", "Matahari & Bulan" at lines 706, 848, 990 of the card-search HTML; "Evolusi Mega" 19 times). Images on these pages are pack/pillow shots (`/id/card-img/products/MA1_pillow_img_IDN.png` etc.), news thumbnails and banners. No Series wordmark file.
- `https://asia.pokemon-card.com/id/archive/special/card/sv/index.html` (200): the Scarlet & Violet campaign page. Its `hero-visual.jpg` (https://asia.pokemon-card.com/id/archive/special/card/sv/assets/images/hero-visual.jpg, 200, `image/jpeg`, 2158x1154) contains the same "Pokemon / Game Kartu Koleksi / Scarlet & Violet" lockup, but it is an opaque JPEG composited over a grey gradient with cards and art, so the wordmark (roughly 450 px wide by 60 px tall at native size, my estimate from the 800 px preview) is no larger than the 398x56 Archives crop and not extractable with alpha. `modal-head-1.png` is a text heading ("Apa itu Pokemon Game Kartu Koleksi?"). Not useful.
- `https://asia.pokemon-card.com/id/archive/special/card/sv1/` (200): Scarlet ex / Violet ex campaign page; `ogp.png` (1200x630) and `hero-visual.jpg` (2880x1540) show set names and art, no Series wordmark. `/archive/special/card/{sw,ss,sc,sc1,sc1a,sc1b,ma,ma1,ma2,me,mega}/index.html` all returned 404. The Mega Evolution campaign pages that exist are `/archive/special/card/ma6/` (200, `logo_*.svg` is the Pokemon TCG lockup, 900x664 viewBox, not a Series logo) and `/archive/special/card/mat/` (200, Deck Taktik).
- `https://asia.pokemon-card.com/id/archive/card/sword_shield_series/SC1A.html` (200): pack image `Booster SC1 setA.png` (650x488 thumb), a product photo with the wordmark printed on the pack. Not a logo.
- `https://www.pokemon.com/id/` and `/id/pokemon-tcg`: 404. `https://tcg.pokemon.com/` redirects to `/en-us/` (200): English only. `https://pokemoncard.co.id/`: DNS does not resolve. `https://www.pokemon-card.com/` (200) is the Japanese site. WebSearch for an AKG Entertainment (Indonesian licensee, `akgentertainment.id`) press kit found only news articles, no downloadable logo kit; unverified whether one exists on request.
- `https://archives.bulbagarden.net/w/api.php` search calls for the three Series (`list=search&srnamespace=6&srsearch=<Series> logo`, `srlimit=50`) returned no Indonesian Series-level file other than the four already known (`SV Era Logo I`, `SS Era Logo I`, `Sword Shield Logo Indonesian`, `Mega Evolution Logo Indonesian`).

## 7. Licence

Every file used or considered carries `{{i-Fairuse-tcg-noncard}}` in its own page wikitext (read via `action=query&prop=revisions&rvprop=content&rvslots=main` for `SV Era Logo I`, `Sword Shield Logo Indonesian`, `Mega Evolution Logo Indonesian`, `First Impact Logo Indonesian`, `MEGA Dream ex Logo Indonesian`, `Scarlet ex Logo Indonesian`, `Shiny VMAX Collection Logo Indonesian`, `SV1 Logo EN`): contributor-claimed fair use of The Pokemon Company artwork, same as the earlier note. No file deviates. The first-party site images are Pokemon Company / Pokemon Card Game Indonesia materials with no stated reuse licence (the pages state none that I read; unverified). Cropping does not change the licence position; the derived crops inherit the same provenance caveat as the existing Matahari & Bulan crop.

## 8. Recommendation

| Series | File to use | Crop box (x,y,w,h) | Output at 80 px delivery height | Change vs today |
|---|---|---|---|---|
| Scarlet & Violet | `SV_Era_Logo_I.png` (https://archives.bulbagarden.net/media/upload/3/3b/SV_Era_Logo_I.png) | 0,193,398,56 | 569x80 (14,502 bytes measured, upscale 1.43x; native 398x56) | Removes the Pokemon / Game Kartu Koleksi lockup; wordmark goes from about 128x18 px to 569x80 in the webp |
| Sword & Shield | `Sword_Shield_Logo_Indonesian.png` | none | 220x80 (unchanged) | None; source is already the best available (1500x545, 6.8x oversampled at 80 px) |
| Mega Evolution | `Mega_Evolution_Logo_Indonesian.png` | optional 6,5,867,308 | 225x80 with trim (220x80 today) | Cosmetic; source (880x320) already the best available |
| Sun & Moon (reference) | `First_Impact_Logo_Indonesian.png` | 0,0,1177,182 (already applied) | 517x80 | None |

Series for which no better source than the current asset exists: Sword & Shield and Mega Evolution (current sources are the maximum on Archives, and nothing first-party is larger). Scarlet & Violet has no better source file either; the improvement is purely the crop, capped at 398x56 native.

Suggested change to `backend/scripts/build-series-logos.sh` (not applied, per the brief): after the fetches, `vips crop "$tmp/scarlet-violet.png" "$tmp/sv-crop.png" 0 193 398 56 && mv "$tmp/sv-crop.png" "$tmp/scarlet-violet.png"`, with a comment that rows 126-170 and 0-122 hold the Game Kartu Koleksi bar and the Pokemon logo and rows 170-193 are the transparent gap.

## 9. Evidence gaps

- The 737x432 older revision of `SV Era Logo I.png` could not be fetched (Cloudflare challenge on every `/media/upload/archive/...` URL, HTTP 403 `cf-mitigated: challenge`). Its content, background colour and whether the wordmark is the same artwork are unverified; the 104 px estimate is a linear scaling guess.
- The hero JPEG wordmark size on the official site is an estimate from a downscaled preview, not a pixel measurement.
- I did not look for, or ask, an AKG Entertainment / Pokemon Indonesia press kit; it could hold vector logos but is unverified. `pokepedia.id` remains off the table (bot-challenged, per the earlier note).
- Archives pages beyond the category and search queries (for example orphaned files with no category and no "logo" in the title) are not covered; the searches above are `srlimit=50`, so a very deep hit list could hide a file.
- Licence reading is of the wiki templates, not legal advice; first-party image terms were not read.
