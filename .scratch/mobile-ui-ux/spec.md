# Mobile UI and UX improvements

Status: resolved

Execution: complete on `feat/mobile-ui-ux` through [PR #95](https://github.com/itsLeonB/cardstack/pull/95). All five implementation tickets are resolved. The user also approved shared E2E session reuse, later review fixes, matching Collection actions, and app motion/loading. See [implementation results](implementation.md) for verification and review outcomes.

## Open follow-up work

PR #95 has merged. The [follow-up spec](../mobile-ui-ux-follow-up/spec.md) and its four tickets are the single source for the remaining second-review work. Start it in a new PR from current `origin/main`; the completed scope below does not include those unverified fixes.

## Problem Statement

Mobile users cannot reach Cardstack’s main tasks through clear, persistent navigation. Guests must open a menu to find Catalog, Log in, and Register. Signed-in users must look inside Account to find their Trades.

Other controls make collection work harder to understand. A Trade shows a raw link instead of a share action. Empty Collections display binder setup before the owner chooses to make a binder. Scan cards and Add Cards lack useful icons. Catalog Cards mix owned-quantity and Wishlist controls, making it hard to tell which quantity an action changes. Large container corners can clip Card artwork.

## Solution

Give mobile users a bottom bar. Signed-in users see Catalog, Collections, Wishlist, Trades, and More. More keeps Master Inventory and Account within reach. Guests see Catalog, Log in, and Register without a menu, with clear calls to action on the home page as well.

Give Trades its own page. Replace the owner’s raw Trade URL with a Share button and popup. Reuse link-delivery behavior across Collection, Wishlist, and Trade callers without changing who can view their links.

Make binder setup an action the owner chooses, not the default content of an empty Collection. Add camera and plus/Card icons to the existing collection actions. Separate Wishlist editing from owned quantity with a labeled button and wanted-quantity popup. Preserve complete Card artwork and its printed corners.

Keep the current brand, house UI Modules, light/dark themes, and all ownership, quantity, and access rules.

## User Stories

### Signed-in mobile navigation

1. As a signed-in mobile user, I want a persistent bottom bar, so that I can reach the main tasks without opening the header menu.
2. As a signed-in mobile user, I want Catalog, Collections, Wishlist, and Trades to stay visible, so that I can switch between the tasks I use often.
3. As a signed-in mobile user, I want More to expose Master Inventory and Account, so that less-used destinations remain easy to find.
4. As a mobile user, I want icons with visible navigation labels and a clear current-page state, so that I do not have to guess what a symbol means or where I am.
5. As a user whose session changes, I want navigation to reflect my current sign-in state without showing another state’s actions while loading, so that the available destinations stay accurate.

### Guest navigation and screen space

6. As a mobile Guest, I want Catalog, Log in, and Register in a visible bottom bar, so that I do not need a menu to discover them.
7. As a Guest on the home page, I want clear Catalog and sign-in/register calls to action, so that I can choose how to start using Cardstack.
8. As a Guest browsing the catalog, I want those three destinations to remain available, so that I can sign in or register without returning home first.
9. As a mobile user, I want bottom navigation to leave content and focused controls visible, so that I can read and operate the full page.
10. As a user scanning Cards, I want the full-screen scan flow to keep working, so that new navigation does not cover the camera or scan controls.

### Trades page

11. As a signed-in user, I want a dedicated Trades page, so that I do not have to look inside Account for exchanges.
12. As a signed-in user, I want to start a Trade from that page, so that creation and the list of my Trades live together.
13. As a participant, I want my existing Trade history in newest-first order, so that I can return to an exchange or inspect its result.
14. As a participant, I want to see the other participant, state, and age of a Trade and cancel it where allowed, so that moving the page does not remove useful behavior.
15. As a user with no Trades or a failed list request, I want a clear empty state or error with Retry, so that I know whether to start a Trade or try loading again.

### Link delivery

16. As a Trade owner, I want a Share button instead of a raw URL on the page, so that the action is clear and the long link does not compete with the Trade content.
17. As a Trade owner, I want Share to open a popup with the link and Copy action, so that I can send the correct Trade link to someone.
18. As a user with a supported browser, I want an explicit action to open native sharing, so that I can send a link through an app on my device.
19. As a user without native-share support, I want Copy to remain available, so that sharing does not depend on one browser feature.
20. As a user whose browser cannot copy automatically, I want a selectable link and clear manual-copy guidance, so that I can still send it.

### Sharing state and privacy

21. As a user who copies a link, I want accessible success or failure feedback on each attempt, so that I know whether the action worked.
22. As a user who cancels native sharing, I want to return to the popup without a false error, so that cancellation does not look like a failed operation.
23. As a Collection or Wishlist owner, I want the same link actions without losing existing share/unshare controls, so that I keep control of read-only access.
24. As a Trade participant, I want link delivery to preserve Trade access rules, so that the Share button does not expose whole holdings or make a Trade guest-readable.
25. As an owner who revokes or replaces a shared link, I want the popup to use the current link and clear old delivery feedback, so that it cannot confirm or deliver an obsolete URL.

### Binder setup

26. As an owner of an eligible empty Collection, I want a secondary “Make this a binder” action, so that binder setup does not replace the page’s normal starting actions.
27. As a Collection owner, I want binder setup to open only when I select that action, so that an empty Collection does not force a setup decision.
28. As a Collection owner, I want to cancel or dismiss binder setup without creating a Layout, so that opening the form does not change my Collection.
29. As a Collection owner, I want the existing Layout choices, defaults, limits, pending state, and inline errors in the dialog, so that moving the form does not change its rules.
30. As a Collection owner, I want a successful creation to show the binder view, so that I can continue without refreshing or seeing stale Collection data.

### Collection actions and Wishlist entry

31. As a Collection owner, I want a camera icon beside Scan cards and a plus/Card icon beside Add Cards, so that I can identify the two actions faster.
32. As a binder owner, I want those actions to keep their existing availability and destinations, so that icons do not introduce a quantity-edit path that ignores Slots.
33. As a signed-in catalog user, I want a separate labeled Wishlist button, so that I can distinguish wanted Cards from the Cards I own in a selected Collection.
34. As a catalog user, I want to see an existing wanted total on or next to that action, so that I can inspect Wishlist state without opening every popup.
35. As a catalog user, I want the button to open wanted quantity and an explicit Save action, so that changing a draft number does not save before I choose to commit it.

### Quantity integrity and artwork

36. As a user editing an existing Wishlist Item, I want the popup to start from its saved wanted total, so that the new presentation does not replace my target with a default of one.
37. As a user editing wanted quantity, I want validation, pending state, and inline save errors, so that I can correct a failed edit without confusing it with an owned-quantity change.
38. As a user who owns a wanted Card, I want the Wishlist Item and its wanted total to remain until I clear it through the existing removal flow, so that fulfilment does not silently delete my target.
39. As a user browsing Card images, I want the full artwork and its own rounded corners to remain visible, so that a UI container does not cut off the printed Card.
40. As a user of another screen that uses the same Card tile, I want its artwork fix without new owner-only actions, so that a shared presentation change does not alter access or controls.

### Accessibility and visual continuity

41. As a keyboard user, I want real links for navigation, visible focus, and usable popup controls, so that I can complete each changed flow without a pointer.
42. As a screen-reader user, I want named actions, correct expanded/current states, named dialogs, and announced results, so that I can understand navigation and saves.
43. As a touch user, I want adequately sized actions and safe-area clearance, so that I can operate them without hitting adjacent controls or device chrome.
44. As a user who zooms text or requests reduced motion, I want readable, operable layouts without unwanted animation, so that the changes respect my display needs.
45. As an existing Cardstack user, I want the current colors, Figtree, Instrument Sans, and both themes, so that these improvements do not replace the product’s visual identity.

## Implementation Decisions

### Scope and existing Modules

- Keep the change in the user-facing frontend. No database schema, backend contract, ownership rule, or quantity model changes are needed.
- Keep existing house UI Modules, semantic theme tokens, Remix icons, Figtree, and Instrument Sans. Do not install a new navigation or overlay library or replace the shadcn preset.
- Keep the existing navigation, Collection, Wishlist, Card tile, and sharing Seams. The only justified shared extraction is link delivery, which has real Collection, Wishlist, and Trade callers.
- Reuse the existing page container and header Modules. Preserve the shell’s single main landmark, route titles, noindex rules, router history, and session-change protections.
- Treat all eight annotations as one agreed feature scope. Keep unrelated cleanup, redesign, and new features out of the implementation.

### Navigation

- Replace the mobile header disclosure with a bottom bar below the existing desktop breakpoint. Use the existing auth-aware destination source rather than maintaining competing route lists.
- Signed-in destinations are Catalog, Collections, Wishlist, Trades, and More. More exposes Master Inventory and Account through the existing overlay primitives.
- Guest destinations are Catalog, Log in, and Register. Remove the guest menu button and retain clear home-page calls to action. Do not require a return to the home page to reach guest navigation.
- Keep the current loading-state rule: show only destinations known to be available until the session resolves. Do not flash guest-only or private actions from the wrong session state.
- Use real router links, visible text, decorative icons, and a current-page indication that does not depend on color alone. Preserve browser Back behavior and existing sign-in return destinations.
- Close More on destination selection or Escape, return focus correctly, and avoid leaving a mobile overlay open after switching to desktop layout.
- Preserve the desktop layout while adding a discoverable Trades destination. Preserve existing footer navigation behavior through the shared destination source.
- Reserve space for the bottom bar and device safe areas so it cannot cover content, focused controls, or long-list status. Preserve the full-screen scan flow rather than placing navigation over its controls.

### Trades page

- Move existing Trade creation, listing, row actions, and cancellation from Account to an authenticated `/trades` page. Account no longer embeds the Trade list or its creation controls.
- Keep the list scoped to the current user, in newest-first order, with existing executed and cancelled history. Keep the existing row content and cancellation eligibility. Cancelled Trades remain non-links.
- Preserve creation navigation to the new Trade, error handling, Retry, empty state, and session-scoped data handling. Use the existing loading primitives where the new page needs a loading state.
- Keep the existing `/trades/:slug` detail route and its distinct access model. Moving the list must not make the detail route inherit a different authentication policy or remove its session-change safeguards and polling.
- Update navigation, titles, breadcrumbs where applicable, and existing flow tests that enter Trade creation or listing through Account. Do not change the Trade state machine or add list filters, pagination, or notifications.

### Link-delivery Module

- Replace the owner-only raw Trade URL with a Share trigger and named popup. Preserve who can see that owner control; do not broaden link access.
- Separate delivery behavior from publication. Collection and Wishlist keep their existing publication controls, confirmation, pending state, and errors. The Trade popup has no share/unshare switch and must not claim that anyone, including a Guest, can view the Trade.
- Place the link-delivery Module at the existing sharing Seam. Its small Interface supplies the current link and needed display text; its implementation owns Copy, native sharing, result feedback, and fallback behavior. Callers retain publication and access decisions.
- Use a selectable current URL, an explicit Copy action, and native sharing when the browser supports it. Start native sharing directly from a user action, not on popup open or after unrelated asynchronous work.
- Detect browser capability on the client. If native sharing is unavailable, retain Copy. If clipboard access is absent or fails, retain manual selection and copy guidance instead of throwing or hiding the link.
- Announce copy outcomes, including repeated attempts. Reset delivery feedback when the link changes or the popup reopens. Treat a native-share cancellation as a non-error; surface other failures without removing the fallback actions.
- Preserve ADR-0025: Collection sharing is opt-in, revocation invalidates the old link, re-sharing creates a new Collection link, and shared views do not expose email or edit controls.
- Preserve ADR-0028: a Trade is not Shared; a Guest cannot read one, a signed-in open-link holder can read offered items under the existing rules, and outsiders lose access once both seats are filled. Link delivery exposes neither a participant’s whole Master Inventory nor private Collections.

### Collection actions and binder dialog

- Show the secondary “Make this a binder” action only under the existing eligibility rule: the owner has a Collection with no Layout and zero Cards. Do not introduce conversion of populated Collections.
- Open the existing binder creation form in the existing Dialog Module only after an explicit click. Never auto-open it on a first visit or empty state.
- Preserve the form’s existing Layout choices, 3-by-3 default, 10-Leaf default, row/column limits of 1–6, and Leaf limits of 1–100. Preserve validation, submission, inline errors, and pending behavior.
- Cancel or dismiss without creating a Layout. After a successful creation, close the dialog and show the updated binder through the existing invalidation/read-back behavior.
- Preserve ADR-0020: Slots determine quantities in a binder; creating its Layout requires an empty Collection and retains the existing capacity-rule change. Do not route binders through bulk Inventory Entry quantity writes.
- Add a decorative camera icon to Scan cards and a plus/Card icon to Add Cards, retaining their visible labels, destinations, and eligibility. Scan stays available for binders; Add Cards does not gain binder availability.

### Wishlist popup

- Change presentation in the existing Wishlist action Module, including its catalog-search and Card-detail callers. Keep owner-only actions outside the shared Card tile Module.
- Render a distinct, labeled Wishlist action apart from the selected Collection’s owned-quantity control. Show an existing saved wanted total on or next to the action.
- Open an anchored popup using the existing overlay primitives. Keep the popup within the viewport on a small Card tile and give its controls clear names and focus behavior.
- Keep the current wanted-quantity draft and explicit save contract. Opening the popup or changing its draft does not write data; the existing add default remains one, and an existing item starts from its saved total. Dismissal without Save does not commit the draft.
- Preserve validation of a positive wanted total, pending state, inline errors, cache updates, and session protections. Saving replaces the wanted total rather than adding to it or changing owned quantity.
- Do not replace Wishlist Items with boolean toggles. Do not add a new removal flow to this popup; retain existing removal behavior on the Wishlist page. Ownership does not delete a Wishlist Item, and a fulfilled item keeps its wanted total until the user clears it.
- Preserve the existing guest sign-in prompt and return destination. Do not render a writable Wishlist action for a Guest or during an unresolved session.

### Card artwork and accessibility

- Fix artwork framing in the shared Card tile Module, not global Card rounding. Reduce or remove container rounding at the image edge as needed and contain, rather than crop, the full image so the printed Card’s own corners remain visible.
- Preserve reserved image dimensions, responsive image sources, loading, and failure behavior. Check the tile’s existing callers without adding owner-only actions to shared views or changing unrelated panel shapes.
- Use visible labels and hide decorative icons from assistive technology. Preserve semantic links for destinations and buttons for actions; avoid nested interactive controls.
- Use named popups/dialogs, correct focus behavior for each overlay, Escape dismissal, and focus return. A modal dialog traps focus and makes its background inert; a non-modal popup must not create a keyboard trap. Announce save and copy results without moving focus unnecessarily.
- Target at least 44-by-44 CSS pixels for the changed mobile actions. Preserve visible keyboard focus, readable contrast in both themes, text zoom, reduced-motion behavior, safe-area clearance, and freedom from page-level horizontal overflow.

## Testing Decisions

### Test Seams and proof ownership

- Prefer the highest existing Seam: rendered route/user flows through the app shell and real router. Verify what users can see, select, navigate to, and save, rather than internal state, CSS class strings, hook calls, or a chosen JSX arrangement.
- Use the existing UI Module Interface only for behavior with meaningful branches that would make end-to-end coverage slow or fragile, such as missing clipboard support, native-share cancellation, and failed saves. The new link-delivery Module is a production extraction for three callers, not a test-only Seam.
- Let focused Module tests own validation and browser-error branches. Let end-to-end tests own changed flows, route/auth integration, and one key failure per flow; do not repeat every Module branch at the browser layer.
- Keep existing backend tests as proof of ownership, Trade access, positive wanted totals, binder eligibility, and Slot-derived quantities. Do not add duplicate backend tests or new dependency-injection options for a frontend presentation change.
- These testing choices come from the existing test structure and code exploration. They do not expand the approved product behavior or authorize implementation.

### Module and route checks

- Navigation: extend the existing SiteHeader and app-shell tests for signed-in, Guest, and resolving-session navigation; current-page state; More selection/Escape/focus; desktop transition; preserved destinations; and sign-in return behavior.
- Trades: move the current list/start/cancel flow assertions to the Trades page. Keep proof that Account no longer contains those controls, only the user’s Trades appear, history remains accessible, and a failed list request offers Retry. Preserve the existing join/exchange and session-change tests rather than restating their full state machine.
- Sharing: extend the existing ShareDialog, CollectionShare, and WishlistShare prior art. Test the link-delivery Interface for Copy success, absent/rejected clipboard, repeated announcements, current-link changes, supported/unsupported native sharing, cancellation, and other failures. Keep caller-level checks that Trade has no publication controls and Collection/Wishlist retain theirs.
- Binder setup: adapt the existing Collection-page and binder-creation tests to click the trigger. Check initial closure, eligibility, cancellation with no write, successful transition, and inline failure. Reuse the existing Layout validation tests instead of testing every limit again through the dialog.
- Collection links: retain existing ScanLink and AddCardsLink destination tests; verify accessible text remains when icons appear. Do not test decorative SVG markup or icon class names.
- Wishlist: adapt the existing catalog search/detail Wishlist tests to open the action before editing. Check saved wanted totals, explicit Save, dismissal without a write, replacement rather than addition, guest prompts, and error recovery. Retain quantity-batching and Wishlist-page removal tests unchanged unless production behavior actually changes.
- Artwork: extend existing Card tile image/fallback coverage only where the rendering contract changes. Use rendered checks for clipping and the full image, not class-name assertions. Sample the existing catalog, Collection, Wishlist, and shared-view callers for unintended layout or action changes.

### Browser verification and tooling

- Extend the current Chromium app-shell and Trade-list browser flows rather than creating a separate navigation test framework. Reuse the existing route test setup, real QueryClient/cache checks where needed, virtual-grid helpers, and API fixtures.
- Follow the repo’s existing Clerk test setup for authenticated end-to-end tests. Do not stub identity or create a second authentication test mechanism.
- Use the shared compose stack and frontend verification entrypoint for lint, formatting, type checking, tests, build, and relevant end-to-end checks. Do not add a host toolchain, database, or browser download.
- Run rendered accessibility checks on the changed flows and exercise keyboard navigation, More, binder setup, Wishlist editing, and sharing. Check small phone widths, landscape, text zoom, both themes, and content/focus clearance around the bottom bar. Automated audit results do not establish WCAG conformance.
- Verify clipboard/native sharing in a suitable secure context. The compose web origin is not secure by default; do not mistake a missing secure-context capability for an implementation failure. Use focused browser-API tests for deterministic branch coverage and report any real-device checks that could not run.
- No browser support policy changed. Report Safari, real native-share, real-device safe-area, or screen-reader checks as untested when the environment cannot perform them.

## Out of Scope

- Production implementation or deployment without a separate instruction to start. The `ready-for-agent` label means the spec is complete, not that execution may begin.
- New backend contracts, database migrations, Trade states, Trade notifications, public inventories, or different link-access rules.
- New Wishlist semantics, boolean Wishlist toggles, automatic removal of fulfilled items, a new popup removal action, or different inventory/binder quantity rules.
- A new design system, shadcn preset, typeface, palette, global Card-radius change, navigation framework, or unrelated accessibility/performance refactor.
- New Trade filtering/pagination, populated-Collection binder conversion, automatic setup prompts, or shipping the prototype’s switcher and sample data.

## Further Notes

The user confirmed the design after choosing bottom navigation for both signed-in users and Guests, a click-open binder dialog, the existing brand, and a separate Wishlist button with a wanted-quantity popup. The user explicitly kept implementation paused. This synthesis requires no further interview.

The original navigation prototype is `.scratch/mobile-ui-ux/prototypes/mobile-nav.html`, captured on `feat/mobile-nav-prototype` at commit `6cf84da`. The decision record is `.scratch/mobile-ui-ux/issues/01-mobile-navigation-prototype.md`. Variant A informed the signed-in layout, but its guest Menu is obsolete: the confirmed guest design uses the three-destination bottom bar. The prototype is a standalone sketch with sample data, not a live app route or production source to copy unchanged.

Prototype verification passed three layouts’ navigation, guest links, Escape/focus return, and query switching; six axe audits covered both themes, and twelve overflow checks covered four widths. These results apply only to the original sketch. They do not verify the revised guest layout, live auth, real quantity changes, native share, or production integration.

The architecture exploration found one justified deepening: common link delivery, separate from publication. Its depth, locality, and leverage come from three real callers sharing browser handling while retaining their own access rules. Keep the other changes local to existing Modules. No new domain terms or ADR changes are needed.

Existing unrelated changes to the skills lockfile and the local modern-web-guidance skill are not part of this feature and must remain untouched.
