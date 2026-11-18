# Mobile UI and UX improvements

Status: needs-info

## Scope

Improve the eight annotated areas: mobile navigation; a dedicated Trades page; Trade link delivery; empty Collection binder setup; Scan cards and Add Cards icons; catalog Wishlist actions; and Card artwork corners. Keep the current colors, Figtree, Instrument Sans, and both themes. Do not change ownership, wanted quantities, sharing access, or Trade rules.

## Decisions

- Open “Make this a binder” only when the user selects it. Do not open the setup dialog automatically on an empty Collection.
- Keep the current visual brand. Change layout, spacing, hierarchy, action placement, icons, and artwork framing only.
- Use bottom navigation for signed-in users. Retain the prototype’s four visible destinations (Catalog, Collections, Wishlist, Trades) plus More for Master Inventory and Account, subject to final brief confirmation.
- Guests also get a bottom bar: Catalog, Log in, Register. Remove the guest menu button and keep clear home-page calls to action.
- Use a separate labeled Wishlist button on catalog Cards. It opens a wanted-quantity popup with an explicit Save action; keep existing wanted totals and save rules.
- The user explicitly said not to implement yet. Record design choices only until they give separate implementation approval.

## Navigation prototype

Question: should mobile navigation use a bottom bar, a side panel, or a compact header row?

Primary source: `feat/mobile-nav-prototype`, `.scratch/mobile-ui-ux/prototypes/mobile-nav.html`.

Open the HTML file directly. `?variant=A`, `?variant=B`, and `?variant=C` choose the starting layout. The arrows change layouts and update the query parameter. Signed in, Guest, and theme buttons change sample state only. Four visible destinations plus More form variant A; all destinations in a side panel form variant B; three header destinations plus More form variant C. The sample includes a dedicated Trades destination and keeps Master Inventory accessible.

This is a standalone visual sketch, not an app route. It uses sample Collections, no live auth, no API calls, no persistent data, and no mutations. Do not promote this file into production. A production choice must use the existing router, house UI Modules, Remix icons, and session rules. Keep this branch as the prototype source and implement the selected design separately after the interview confirms the scope.

Verdict: the user selected bottom navigation. They changed the guest design to a three-destination bottom bar, not the Menu shown in the original prototype. Keep the original artifact as a record; it does not show the final guest decision.

## Code facts

- Navigation comes from `frontend/src/components/layout/nav-links.ts` and `site-header.tsx`. Account owns Trade creation, list, and cancellation today. Move that behavior to an authenticated `/trades` index; preserve the existing `/trades/$slug` access rules and history.
- `frontend/src/components/share/share-dialog.tsx` already serves Collections and Wishlist. Share/unshare controls must stay separate from link delivery: a Trade is not Shared. A common link-delivery Module can own Copy, native share, feedback, and fallbacks for all three callers without changing publication.
- Empty binder setup comes from `CreateBinderPanel`, which already owns pending state, errors, and invalidation. `BinderLayoutForm` already supports cancellation. Keep the backend’s empty-Collection restriction and Slot-derived quantities.
- `ScanLink` and `AddCardsLink` have one production caller each and can gain decorative icons directly.
- Wishlist saves a positive wanted total, not an owned quantity. A boolean toggle loses a prior wanted total when an item is removed and re-added. Retain arbitrary wanted quantities unless the user explicitly chooses a different model.
- `CardTile` controls artwork framing across many screens. Change it instead of global `ui/Card` rounding, and keep owner-only actions out of it.

## Proposed final brief

1. Mobile navigation: signed-in Catalog, Collections, Wishlist, Trades, More; More keeps Master Inventory and Account. Guests get Catalog, Log in, Register with no menu. Keep home-page auth calls to action. Keep desktop layout, add a Trades destination, and preserve the full-screen scanning flow.
2. Trades: move creation, list, cancellation, and full history out of Account into authenticated `/trades`. Keep `/trades/$slug` access rules. Replace the owner’s raw URL with Share and a popup. Reuse link delivery across Collection, Wishlist, and Trade; keep publication controls out of the Trade popup. Keep Copy and manual-copy fallback; offer native share when available, only on a user click. Treat native cancellation as a non-error.
3. Collections: use a secondary “Make this a binder” action on eligible empty Collections to open the existing form in a Dialog. Never auto-open it. Preserve limits, errors, pending behavior, and cancellation. Add decorative camera and plus/Card icons to Scan cards and Add Cards without changing destinations.
4. Catalog: separate Wishlist from owned quantity. Open the wanted-quantity popup from a labeled button and keep explicit Save. Preserve arbitrary wanted totals and all inventory behavior. Fix artwork framing in CardTile so the full image and its own corners remain visible, without changing global Card rounding.
5. Accessibility and checks: retain house UI Modules, existing tokens, Figtree, Instrument Sans, and both themes. Use labeled controls and decorative icons, 44 CSS-pixel mobile targets, visible focus, correct popup focus behavior, reduced-motion support, and safe-area/content clearance. Test the changed flows with the compose verification scripts and rendered checks; report any untested browser or assistive technology.

## Architecture finding

Read-only exploration found one justified shared Module: link delivery, separate from publication. Collection, Wishlist, and Trade are real callers. Keep the other changes inside existing Modules; no new backend Adapter, navigation framework, quantity abstraction, or theme system. This respects ADR-0025 and ADR-0028. The temporary visual report is `/tmp/architecture-review-20261118-122239.html`; the facts above remain the durable record.

## Open decisions

- Confirm the proposed final brief as the shared understanding. This confirms design only; it does not authorize implementation.
- Obtain separate permission before production edits.

## Verification

Prototype checks run in the existing compose `frontend-tools` container with Playwright and axe. Record the final results in the prototype issue. These checks do not prove the live app works or establish WCAG conformance. No live-page audit, Safari test, or screen-reader test has run.
