# 04: Add mobile bottom navigation

**What to build:** Let mobile users reach Cardstack’s main destinations through a visible bottom bar, with the correct destinations for signed-in users and Guests and no content hidden beneath it.

**Blocked by:** 03 — Give Trades its own page.

**Status:** ready-for-agent

**Execution:** paused. The user approved the ticket breakdown, not implementation. Wait for a separate instruction to start.

## Acceptance criteria

### Destinations and session state

- [ ] Below the existing desktop breakpoint, signed-in users see Catalog, Collections, Wishlist, Trades, and More. More exposes Master Inventory and Account.
- [ ] Guests see Catalog, Log in, and Register in the bottom bar on guest pages. Remove the guest menu button and retain clear home-page Catalog/sign-in/register calls to action.
- [ ] Use the existing auth-aware navigation source. While the session resolves, show only known-available destinations; do not flash actions from the wrong auth state.
- [ ] Use real router links, visible labels, decorative Remix icons, and a current-page indication that does not depend on color alone. Keep Back behavior and existing sign-in return destinations.
- [ ] More uses the existing overlay primitives, closes on selection or Escape, returns focus correctly, and does not stay open after a desktop transition. Preserve desktop and footer navigation from ticket 03.

### Screen space and accessibility

- [ ] Reserve space for the bottom bar and device safe areas. Do not cover content, the focused control, virtualized-list status, or the last reachable action; preserve useful scrolling without page-level horizontal overflow.
- [ ] Preserve the full-screen scanning flow so the bar does not cover camera or scan controls. Do not redesign scanning as part of this ticket.
- [ ] Keep visible focus, correct current/expanded states, named controls, and at least 44-by-44 CSS-pixel touch targets for the changed mobile actions.
- [ ] Preserve existing colors, Figtree, Instrument Sans, both themes, text zoom, and reduced-motion behavior. Add no new navigation framework, theme system, or overlay dependency.
- [ ] Use the confirmed guest design rather than copying the prototype’s obsolete Guest Menu. Do not ship sample data or the prototype switcher.

## Verification

- [ ] Extend existing SiteHeader and app-shell prior art for signed-in, Guest, and resolving-session destinations; active state; More navigation/Escape/focus; sign-in return behavior; and transition to desktop.
- [ ] Exercise a real route flow to the Trades page from ticket 03, plus guest Catalog and auth destinations. Use the existing Clerk and router test setup, not fake identity.
- [ ] Run rendered accessibility and keyboard checks at small phone widths, landscape, text zoom, and both themes, including scan/content clearance. Run relevant compose frontend verification checks and report any real-device safe-area or assistive-technology checks that could not run.

## Scope limits

Ticket 03 is the only ticket dependency: the Trades destination must exist. This ticket does not depend on the Share, binder, or catalog changes. No desktop redesign, prototype promotion, or unrelated navigation cleanup. Read the feature’s approved design spec before implementation.
