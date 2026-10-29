# 10: Share the Wishlist

**What to build:** a user shares their Wishlist with the same Share dialog as a Collection, and anyone, a Guest included, opens `/s/<profile slug>/wishlist` and sees the owner's items with their owned and wanted and a status they can filter by (GLOSSARY.md: Shared, Wishlist; ADR-0025).

**Blocked by:** 02 (Share a Collection and open its link), 07 (Wishlist page).

**Status:** ready-for-agent

- [ ] A Wishlist has a `shared` flag, off by default. The Share dialog from ticket 02 is reused on the Wishlist page with the same warning, switch, link, Copy and unshare confirmation, and turning sharing off ends the link at once.
- [ ] The public route group serves a Shared Wishlist: the owner's display name, each item's Card, wanted quantity, the owner's owned quantity and the Fulfilled or Still needed status, with the shared card filters and the All, Still needed and Fulfilled filter available to Guests. It never returns the owner's email.
- [ ] A private, unshared or nonexistent Wishlist gives the same identical not-found as a Collection. The page carries `noindex` and has no edit controls, no messaging and no contact details.
- [ ] HTTP-boundary tests cover share, unshare and the Guest view. Screen tests cover the Wishlist's Share dialog and the shared page.
