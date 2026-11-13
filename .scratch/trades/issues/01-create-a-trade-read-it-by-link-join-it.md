# 01: Create a Trade, read it by link, join it

**What to build:** A signed-in user starts a Trade and gets a link to it. Another signed-in user opens that link, reads what the first side is offering, and joins, taking the second seat. A third opener is told the Trade is full; a Guest, an unknown link and a cancelled one all get the same not found. This is the room existing and two people meeting in it: no Trade Items yet, nothing movable.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Creating a Trade puts the caller on one side in state `open` and returns the Trade's slug.
- [x] Reading the Trade by slug returns its state, both participants' display names, and the Trade Items on each side (empty for now) without claiming anything.
- [x] Joining fills the second seat once and moves the Trade to `negotiating`.
- [x] A second join attempt fails with a stable seat-taken code, and two joins issued at once leave exactly one winner.
- [x] A Guest opening a Trade link gets the same not-found answer as an unknown slug.
- [x] A signed-in user who is not a participant gets that same not-found answer.
- [x] No email address or Auth Identity appears in any Trade payload, only display names.
- [x] Trade pages are excluded from search engines.
- [x] Routes tests cover creation, read-before-join leaving the seat free, the join race, and each refusal. The Trade page renders by slug and joins in the browser.
