# 02: Share links from Collections, Wishlist, and Trades

**What to build:** Let a Trade owner open a Share popup instead of reading a raw URL on the page. Give Collection, Wishlist, and Trade callers the same reliable Copy and native-share actions while keeping their distinct publication and access rules.

**Blocked by:** None (can start immediately after implementation approval).

**Status:** resolved

**Execution:** approved by the user's implement-spec request; integration branch `feat/mobile-ui-ux`.

## Acceptance criteria

### User flow

- [ ] The existing owner-only Trade link becomes a labeled Share button that opens a named popup with the current, selectable URL. Do not broaden who sees the control.
- [ ] Copy works when clipboard access is available and announces success or failure on each attempt, including repeated copies of the same link.
- [ ] If clipboard access is absent or rejects the request, keep the selectable link and clear manual-copy guidance. Do not throw or remove the remaining actions.
- [ ] Offer native sharing only when the client browser supports it, start it directly from an explicit user action, treat cancellation as a non-error, and leave Copy/manual copy available after other failures.
- [ ] Reset delivery feedback when the popup reopens or its link changes, so feedback cannot describe an obsolete or revoked link.

### Shared behavior and access

- [ ] Extract the link-delivery behavior from the existing sharing Module before connecting the Trade caller. Collection and Wishlist use the same production Interface; do not add a forwarding-only Module or test-only configuration.
- [ ] Collection and Wishlist retain their publication controls, pending/error states, name warning, revocation confirmation, and current-link behavior. Keep publication decisions outside link delivery.
- [ ] The Trade popup has no publication switch or wording that claims a Guest can view it. Preserve ADR-0025 and ADR-0028: a Trade is not Shared, only offered items become visible, and no whole holdings or private Collections become visible.
- [ ] Preserve the existing detail route, owner rules, session-change protections, polling, and Trade state behavior. This ticket does not depend on moving the Trade list out of Account.
- [ ] Reuse the current house overlay/button/input Modules, semantic theme tokens, and Remix icons. Preserve both themes, named controls, announced results, Escape/focus return, and 44-by-44 CSS-pixel changed mobile actions.

## Verification

- [ ] Extend the existing share-dialog and Collection/Wishlist sharing prior art. Test observable delivery behavior for supported/unsupported native sharing, cancellation, other share failures, clipboard success/absence/rejection, repeated announcements, and changed links. Avoid assertions on internal state or CSS classes.
- [ ] At the caller Seam, verify that Collection/Wishlist keep publication controls while Trade never gains them. Keep existing backend access tests rather than duplicating them for this frontend change.
- [ ] Exercise the real popup flow with keyboard and rendered accessibility checks, and run the relevant compose frontend verification checks. Report real native-share or secure-context checks that cannot run; the compose web origin is not secure by default.

## Comments

Implementation commits: `985c12bf`, `930fd3c7` on `t3/1631c7aa`. Both apps passed lint, format, typecheck, unit tests, and builds. Full E2E reported 74 passed and 18 failed; the Trade owner popup flow passed. Merge waits for failed-check diagnosis and a passing final gate. Real-device, Safari, screen-reader, and secure-origin native-share checks did not run.

## Answer

Merged in `61ebad6b`, including verified worker tip `eb40d5d4`. Collection, Wishlist, and Trade now share link delivery while retaining their publication and access rules. Both apps passed all verification steps on the final combined source: 889 frontend unit tests, 108 Admin unit tests, and 94 E2E tests. Sixteen focused sharing E2E checks passed named dialogs, keyboard dismissal, focus return, and accessibility in both themes. Worker worktree and branch were removed safely. Real OS native sharing, Safari, real-device, and screen-reader checks did not run.

## Scope limits

No backend/schema change, new share permissions, new overlay library, Trade notifications, or change to which Trade links work. Read the feature’s approved design spec before implementation.
