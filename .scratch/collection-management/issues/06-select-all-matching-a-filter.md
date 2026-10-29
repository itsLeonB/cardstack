# 06: Select all matching a filter

**What to build:** in selection mode, when the current filter matches more than one page, a banner offers "Select all N matching this filter". Choosing it moves every matching card across all pages, so a user can filter a Collection that holds MA5 and MA6 by MA5 and move all of it.

**Blocked by:** 04 (Move selected cards between plain Collections).

**Status:** ready-for-agent

- [ ] The Move operation also accepts the shared card-filter parameters instead of a list of Card ids, acting on every matching entry of the source across all pages. The two forms are mutually exclusive.
- [ ] The banner appears after the page's cards are selected and the filter has more results than the loaded pages, shows the total N, and replaces the ticked cards with the filter. Changing the filter or leaving selection mode clears it.
- [ ] The confirm dialog states the cards and copies the filter selects, as counted by the server.
- [ ] HTTP-boundary tests cover a Move by filter against a Collection holding two sets, including filters combined with name search, and that ids and filter together are refused. Screen tests cover the banner and its reset.
