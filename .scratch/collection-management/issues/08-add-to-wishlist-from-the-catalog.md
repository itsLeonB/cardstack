# 08: Add to Wishlist from the catalog

**What to build:** a user adds a Card to their Wishlist with a wanted quantity from where they find it: the catalog card page and the catalog search results.

**Blocked by:** 07 (Wishlist page).

**Status:** ready-for-agent

- [ ] An "Add to Wishlist" action with a quantity stepper that starts at 1 sits on the catalog card page and on each catalog search result, for signed-in users only. A Guest is sent to sign in.
- [ ] Adding a Card already on the Wishlist replaces its wanted quantity instead of creating a second item, and the action shows when a Card is already wanted and with what quantity.
- [ ] The result is reflected on the Wishlist page, and the action's outcome is announced.
- [ ] Screen tests with the network mocked cover adding, replacing and the Guest case.
