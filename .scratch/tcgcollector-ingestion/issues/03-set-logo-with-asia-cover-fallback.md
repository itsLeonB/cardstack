# 03: Expansion Set logo with the Asia cover as fallback

**What to build:** An Expansion Set can carry a logo. Where the API returns a set's cover, it returns the logo when the set has one and the existing Pokémon Asia cover otherwise, so users see logos wherever they exist and nothing changes elsewhere. The logo is stored like the other third-party images: the scraped address is kept, and `host-images` rehosts it to R2 (ADR-0016). The logo is hosted as is, not pre-sized (ADR-0017 is unchanged for covers).

**Blocked by:** None (can start immediately).

**Status:** resolved

- [ ] A migration adds `source_logo_url` and `logo_key` to `expansion_sets`, both empty by default
- [ ] Every API response that gives a set's cover URL gives the logo URL when `logo_key` is set and the existing cover otherwise
- [ ] `host-images` hosts a set's logo from `source_logo_url` into `logo_key`, skips sets that already have one, and is safe to rerun
- [ ] Handler tests show a set with a logo returns the logo, a set without returns the cover, and a set with neither returns what it returns today
- [ ] A host-images test shows the logo is hosted and the key is saved
- [ ] The frontend needs no code change; note in the ticket comments any set-list layout concern for a later check
- [ ] Backend verification script passes
