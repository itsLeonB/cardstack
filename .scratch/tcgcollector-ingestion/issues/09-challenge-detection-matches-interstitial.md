# 09: Challenge detection matches the interstitial, not Cloudflare's scripts

**What to build:** A live run never stalls asking the operator to solve a challenge nobody is being shown. The browser fetcher reads a page as challenged only when Cloudflare's interstitial is actually on screen — its challenge config key or its `Just a moment` title — and reads Cloudflare's precursor script and `window.__CF$cv$params` block as ordinary page content, so a normal page goes straight to the parsers. When an interstitial genuinely appears, the pause-and-resume behaviour of ticket 07 is unchanged, and nothing is deleted from the page to get past it (ADR-0021).

**Blocked by:** None (ticket 07 is resolved; this narrows what it detects).

**Status:** ready-for-agent

- [ ] A trimmed copy of a normal saved page carrying the precursor script and the `window.__CF$cv$params` block is not read as a challenge
- [ ] A trimmed interstitial page is still read as a challenge, and the run still pauses and resumes once it clears
- [ ] The fetched page is returned unmodified: no script stripping, no DOM editing, no extra page evaluation
- [ ] Cached pages still hold the page as the browser rendered it
- [ ] Backend verification script passes
- [ ] A manual check is recorded in this ticket: one live dry run on a single Expansion Set completes without asking for a challenge
