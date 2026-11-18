# Choose mobile navigation

Type: prototype

Status: ready-for-human

## Question

Should mobile users navigate through a bottom bar, a side panel, or a compact header row?

## Source

Branch: `feat/mobile-nav-prototype`.

Prototype: `../prototypes/mobile-nav.html`. Open it directly; no server is needed. Query variants: `?variant=A`, `?variant=B`, `?variant=C`. The interface switches between all three layouts, signed-in/Guest sample state, and light/dark themes. The query parameter records the layout only. State is in memory.

This standalone sketch departs from the preferred live-route prototype: it lets the interview compare navigation without touching real auth, app routes, or data. Its content is sample data, not an account. It does not prove integration with the live app. No production code or dependencies changed.

## Recommendation

A: bottom navigation with Catalog, Collections, Wishlist, Trades, and More. More keeps Master Inventory and Account available. The user has not approved this ordering or the shorter visible labels. Guests use a separate Menu.

## Checks

Playwright and axe ran in the existing compose `frontend-tools` container. All three variants passed destination switching, guest links, Escape dismissal, return of focus, and query-parameter switching. Six axe audits covered each variant in both themes with no reported violations. Twelve width checks covered 320, 375, 390, and 728 CSS pixels with no page-level horizontal overflow. The initial run found low contrast in the prototype’s outer notes; explicit note surfaces fixed it before the passing run.

Not run: live app integration, frontend verification scripts, Safari, real-device safe areas, or a screen reader. Automated audit results do not establish WCAG conformance.

## Answer

Pending the user’s selection. Keep the prototype on its own branch. Implement the selected design only after the design interview confirms the final brief.
