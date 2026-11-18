# Mobile UI and UX improvements

Status: needs-info

## Scope

Improve the eight annotated areas: mobile navigation; a dedicated Trades page; Trade link delivery; empty Collection binder setup; Scan cards and Add Cards icons; catalog Wishlist actions; and Card artwork corners. Keep the current colors, Figtree, Instrument Sans, and both themes. Do not change ownership, wanted quantities, sharing access, or Trade rules.

## Decisions

- Open “Make this a binder” only when the user selects it. Do not open the setup dialog automatically on an empty Collection.
- Keep the current visual brand. Change layout, spacing, hierarchy, action placement, icons, and artwork framing only.
- Prototype mobile navigation before choosing its layout. The user has not approved a navigation variant or production implementation.

## Navigation prototype

Question: should mobile navigation use a bottom bar, a side panel, or a compact header row?

Primary source: `feat/mobile-nav-prototype`, `.scratch/mobile-ui-ux/prototypes/mobile-nav.html`.

Open the HTML file directly. `?variant=A`, `?variant=B`, and `?variant=C` choose the starting layout. The arrows change layouts and update the query parameter. Signed in, Guest, and theme buttons change sample state only. Four visible destinations plus More form variant A; all destinations in a side panel form variant B; three header destinations plus More form variant C. The sample includes a dedicated Trades destination and keeps Master Inventory accessible.

This is a standalone visual sketch, not an app route. It uses sample Collections, no live auth, no API calls, no persistent data, and no mutations. Do not promote this file into production. A production choice must use the existing router, house UI Modules, Remix icons, and session rules. Keep this branch as the prototype source and implement the selected design separately after the interview confirms the scope.

Verdict: pending user selection.

## Code facts

- Navigation comes from `frontend/src/components/layout/nav-links.ts` and `site-header.tsx`. Account owns Trade creation, list, and cancellation today. Move that behavior to an authenticated `/trades` index; preserve the existing `/trades/$slug` access rules and history.
- `frontend/src/components/share/share-dialog.tsx` already serves Collections and Wishlist. Share/unshare controls must stay separate from link delivery: a Trade is not Shared. A common link-delivery Module can own Copy, native share, feedback, and fallbacks for all three callers without changing publication.
- Empty binder setup comes from `CreateBinderPanel`, which already owns pending state, errors, and invalidation. `BinderLayoutForm` already supports cancellation. Keep the backend’s empty-Collection restriction and Slot-derived quantities.
- `ScanLink` and `AddCardsLink` have one production caller each and can gain decorative icons directly.
- Wishlist saves a positive wanted total, not an owned quantity. A boolean toggle loses a prior wanted total when an item is removed and re-added. Retain arbitrary wanted quantities unless the user explicitly chooses a different model.
- `CardTile` controls artwork framing across many screens. Change it instead of global `ui/Card` rounding, and keep owner-only actions out of it.

## Open decisions

- Pick a mobile navigation layout, then settle its visible destinations and overflow placement.
- Choose the catalog Wishlist action and wanted-quantity editing flow.
- Confirm Trade history presentation and link-delivery behavior without publication controls.
- Confirm artwork framing, binder entry point, and accessibility acceptance checks.
- Review the scoped architecture candidates, then confirm the final implementation brief before production edits.

## Verification

Prototype checks run in the existing compose `frontend-tools` container with Playwright and axe. Record the final results in the prototype issue. These checks do not prove the live app works or establish WCAG conformance. No live-page audit, Safari test, or screen-reader test has run.
