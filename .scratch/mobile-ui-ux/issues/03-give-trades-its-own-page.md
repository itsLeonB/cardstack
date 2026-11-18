# 03: Give Trades its own page

**What to build:** Let a signed-in user find, start, and manage Trades on a dedicated Trades page instead of inside Account, without losing existing history or behavior.

**Blocked by:** None (can start immediately after implementation approval).

**Status:** claimed

**Execution:** approved by the user's implement-spec request; integration branch `feat/mobile-ui-ux`.

## Acceptance criteria

### Page and navigation

- [ ] Add an authenticated `/trades` page that shows the current user’s Trades and existing creation action. Account no longer embeds the Trade list or creation controls.
- [ ] Expose Trades through the existing shared navigation destination source so the page is reachable before the bottom-bar ticket lands. Preserve the current desktop layout and footer behavior.
- [ ] Use the existing page container/header Modules, route title, appropriate private-page indexing behavior, and router navigation. Preserve the shell’s single main landmark and sign-in return behavior.
- [ ] Show a useful loading state, the existing empty-state direction, and an inline list error with Retry rather than leaving an unexplained blank page.
- [ ] Starting a Trade navigates to its detail view; creation and allowed cancellation keep their current pending/error handling and list updates.

### History and access

- [ ] Keep the list scoped to the current user, newest first, including executed and cancelled history. Preserve the other participant, state, age, and current cancellation eligibility; cancelled rows remain non-links.
- [ ] Keep `/trades/:slug` as the existing distinct detail route. Do not move it under a different auth guard or change its polling, open-link rules, participant access, or session-change safeguards.
- [ ] Retain user-scoped cache clearing and rejection of stale responses across session changes. Moving the list must not expose a previous user’s data.
- [ ] Preserve ADR-0028: the Trade list remains the way to return to a Trade; this change adds no notifications or public inventory.
- [ ] Keep both themes, visible focus, semantic links/actions, and accessible names. Every changed mobile action targets at least 44-by-44 CSS pixels.

## Verification

- [ ] Adapt existing Trade-list/start/cancel tests to enter through the new page and assert Account no longer contains those controls. Cover successful creation/listing and a failed list request with Retry without repeating every Trade-state branch.
- [ ] Update the existing Trade join/exchange and navigation flows that start from Account. Keep their existing proof of access, cancellation, and execution instead of adding a second test framework.
- [ ] Run relevant compose frontend verification checks and a rendered keyboard/accessibility check of the page. Report any authenticated checks that cannot run with the available Clerk setup.

## Scope limits

No new Trade states, filtering, pagination, notifications, database/API changes, or dependency on the Share popup. Read the feature’s approved design spec before implementation.
