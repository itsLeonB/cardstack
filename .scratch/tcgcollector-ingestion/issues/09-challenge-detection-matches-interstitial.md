# 09: Challenge detection matches the interstitial, not Cloudflare's scripts

**What to build:** A live run never stalls asking the operator to solve a challenge nobody is being shown. The browser fetcher reads a page as challenged only when Cloudflare's interstitial is actually on screen — its challenge config key or its `Just a moment` title — and reads Cloudflare's precursor script and `window.__CF$cv$params` block as ordinary page content, so a normal page goes straight to the parsers. When an interstitial genuinely appears, the pause-and-resume behaviour of ticket 07 is unchanged, and nothing is deleted from the page to get past it (ADR-0021).

**Blocked by:** None (ticket 07 is resolved; this narrows what it detects).

**Status:** done — on `fix/challenge-detection-interstitial`, commits `dbc7001`, `7d205bc`, `a3d253a`

- [x] A trimmed copy of a normal saved page carrying the precursor script and the `window.__CF$cv$params` block is not read as a challenge
- [x] A trimmed interstitial page is still read as a challenge, and the run still pauses and resumes once it clears
- [x] The fetched page is returned unmodified: no script stripping, no DOM editing, no extra page evaluation
- [x] Cached pages still hold the page as the browser rendered it
- [x] Backend verification script passes
- [x] A manual check is recorded in this ticket: one live dry run on a single Expansion Set completes without asking for a challenge

**Manual check (2026-10-17):** `make ingest-tcgcollector ARGS="-set MP -refresh"` fetched Mega Evolution Promos (set 11922) from the live site through the browser fetcher, `-refresh` so every page came from the network instead of the page cache. No challenge was detected and the run never printed the solve-it-in-Chrome warning; the operator stopped it early once it was clearly past the pages that used to stall. The `-refresh` flag matters: without it the run reads cached pages and never exercises detection.

**Implementation notes:** `challengeMarkers` is now `{"_cf_chl_opt", "<title>just a moment"}`. `challenge-platform` and the bare `cf_chl_opt` substring are gone, because Cloudflare's precursor script and its `window.__CF$cv$params` block ride on every normal page and both markers matched them. `TestIsChallenge` pins each marker on its own — `configKeyPage` carries the config key under a normal title, and a title-only page carries nothing else — so deleting either marker fails a subtest. `normalPage` is the `window.__CF$cv$params` block and precursor script taken verbatim from the saved sets page in `references/tcg-collector/`; no saved interstitial exists, so `interstitialPage` is written down rather than captured, and its comment says so.
