# 04: Filter and delete match history

**What to build:** Let a user find matches in their private history and permanently remove a mistaken match without changing their Decks or other records.

**Blocked by:** 02: Record, resume and finish matches.

**Status:** ready-for-agent

- [ ] Show the owner's history newest first by played date and filter by their personal Deck and In Progress or Finished status. Apply the filters together, preserve the selected personal Deck's identity when builds share Key Pokémon, and show a clear empty result.
- [ ] Open a match from filtered history to view it or resume it when In Progress. Keep its saved historical Deck details rather than replacing them with the current Deck name or Key Pokémon.
- [ ] Require confirmation before the user permanently deletes a match and its Match Games. Cancelling the confirmation changes nothing; confirmed deletion removes the match from history and leaves its personal Deck, other matches and owned quantities unchanged.
- [ ] Apply owner isolation to history, filters, individual records and deletion, including direct requests. When archived Decks become available, their matches remain discoverable; do not require the completed-match correction feature to implement this slice.
- [ ] Automated checks cover combined filters and ordering, isolation, persisted deletion and preservation of unrelated data. A browser journey filters history and cancels then confirms deletion. Do not add statistics, extra filters, sharing or an opponent library.
