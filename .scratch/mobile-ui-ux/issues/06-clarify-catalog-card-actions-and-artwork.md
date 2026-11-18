# 06: Clarify catalog Card actions and artwork

**What to build:** Let a user distinguish wanted quantity from owned quantity when browsing Cards. Put Wishlist editing behind a clear labeled action and show complete Card artwork without clipping its printed corners.

**Blocked by:** None (can start immediately after implementation approval).

**Status:** ready-for-agent

**Execution:** paused. The user approved the ticket breakdown, not implementation. Wait for a separate instruction to start.

## Acceptance criteria

### Wishlist editing flow

- [ ] Replace the crowded inline Wishlist editor with a separate labeled Wishlist button in its existing catalog-search and Card-detail callers. Keep it distinct from a selected Collection’s owned-quantity control.
- [ ] Show the saved wanted total on or next to the action for an existing Wishlist Item. Keep owner-only actions outside the shared Card tile Module.
- [ ] Clicking opens a named anchored popup through the existing overlay primitives. Keep the popup within the viewport on narrow Card tiles and preserve keyboard operation, Escape dismissal, and focus return without a keyboard trap.
- [ ] Start a new item with the existing default of one and an existing item with its saved wanted total. Opening or changing a draft causes no write; an explicit Save commits it, and dismissal without Save does not.
- [ ] Preserve positive-total validation, pending state, inline errors, cache updates, and session protections. Save replaces the wanted total rather than adding to it or changing owned quantity.

### Quantity integrity and artwork

- [ ] Do not use boolean Wishlist toggles, add a popup removal flow, or remove a fulfilled item automatically. Preserve the existing Wishlist-page removal flow and arbitrary wanted totals.
- [ ] Preserve Guest sign-in prompts and return destinations. Do not expose a writable action to a Guest or during an unresolved session, and do not change owned-quantity batching or binder quantity rules.
- [ ] Fix framing inside the existing shared Card tile Module, not global Card rounding. Reduce or remove container rounding at the image edge as needed and contain the full image so the printed Card’s own corners remain visible.
- [ ] Preserve reserved image dimensions, responsive image sources, loading, and failure behavior. Existing catalog, Collection, Wishlist, and shared-view callers keep their own controls and access rules.
- [ ] Keep the current brand, both themes, visible labels/focus, decorative icons, accessible popup/save feedback, reduced-motion behavior, and at least 44-by-44 CSS-pixel changed mobile actions. Do not add dependencies or redesign unrelated Card panels.

## Verification

- [ ] Adapt existing catalog search/detail Wishlist tests to open the popup. Cover initial saved totals, explicit Save, dismissal without a write, replacement rather than addition, Guest prompts, and error recovery. Reuse existing quantity-batching and Wishlist removal tests unless their production contract changes.
- [ ] Extend Card tile image/fallback tests only where observable rendering behavior changes. Verify clipping and the full image in rendered checks rather than asserting CSS class strings.
- [ ] Exercise a browse-and-edit flow with keyboard, both themes, narrow widths, and rendered accessibility checks. Sample other Card tile callers for unwanted clipping or new owner-only actions, then run relevant compose frontend verification checks.

## Scope limits

No new Wishlist semantics, ownership model, removal behavior, backend/schema changes, global Card radius, image-hosting contract, or dependency on sharing, binder setup, or bottom navigation. Read the feature’s approved design spec before implementation.
