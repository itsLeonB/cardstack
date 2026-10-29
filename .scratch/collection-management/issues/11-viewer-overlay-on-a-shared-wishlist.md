# 11: Viewer overlay on a Shared Wishlist

**What to build:** a signed-in person who opens someone else's Shared Wishlist sees how many of each Card they own themselves and can filter by it, so they can see where a trade is possible.

**Blocked by:** 10 (Share the Wishlist).

**Status:** ready-for-agent

- [ ] For a signed-in viewer, the public Wishlist read also returns the viewer's own owned quantity per item from their Master Inventory, and supports an "I have it" and "I do not have it" filter. The viewer's data is used for that request only and is never returned to the owner.
- [ ] A Guest gets no overlay and no such filter, and sees a "Sign in to see what you own" banner. The owner opening their own Shared link sees their own Wishlist with no overlay.
- [ ] The page shows "you own N" next to each item and the extra filter for a signed-in viewer, alongside the owner's progress and status filter, and no messaging or offer flow.
- [ ] HTTP-boundary tests cover the overlay and filter for a signed-in viewer, their absence for a Guest and for the owner, and that no viewer data reaches the owner. Screen tests cover the three views.
