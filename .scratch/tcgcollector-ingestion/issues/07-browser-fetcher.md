# 07: Browser fetcher that survives the challenge

**What to build:** Without `-dir`, the command fetches pages through a visible Chrome the operator unlocks once. It keeps a persistent profile so later runs start unlocked, waits and resumes when the challenge reappears, and spaces requests like a person would. It does nothing to evade the challenge (ADR-0021): no headless mode, no stealth patches, no solvers.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] A page-source implementation attaches to a visible Chrome using a persistent profile directory and returns the page HTML for a URL
- [ ] Requests run one at a time, a few seconds apart, and never use a `?page=` URL
- [ ] When the challenge page appears, the command prints a message, waits, and continues from the same URL once the operator has solved it; it does not retry in a loop or use any bypass
- [ ] The fetcher is the default source when `-dir` is not given, and plugs under the cache-backed source from ticket 06 once that lands
- [ ] A short doc line says how to start Chrome, unlock it once, and fall back to saving pages for `-dir`
- [ ] Not tested in CI; the ticket comments record a manual check: unlock, fetch the sets page, fetch a card list page, and resume after a forced re-challenge
- [ ] Backend verification script passes
