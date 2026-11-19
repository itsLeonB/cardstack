# 02: Finish Catalog ordering and styled dropdowns

**What to build:** Catalog shows Expansion Sets in its real newest-release-first order. Collection and binder controls use the existing house dropdown pattern, with readable choices and usable spacing in both themes.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

Follow-up to [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 08. Saved changes have no source commit and lack final proof. Read the [saved-work handoff](../handoff.md) before changing them.

## Acceptance criteria

- [ ] User-visible Expansion Set options match the Catalog's actual release-date-descending contract, including grouping and stable ties, with unknown dates last. Use real metadata, not dates guessed from names or codes; do not change Card search order.
- [ ] Cover that order with a deliberately unsorted release-date fixture, including unknown dates and ties.
- [ ] The Add to Collection popup uses the existing house styled dropdown, with readable contrast in light and dark themes and enough internal space for its chevron.
- [ ] Preserve None, current selection, labels, disabled/loading/error states, Collection request parameters, Guest limits, ownership and binder restrictions.
- [ ] Use the same existing pattern for the other app-owned non-Trade native dropdowns, including binder presets. Preserve preset/custom transitions, form values and validation; leave Admin, Clerk controls and native file pickers unchanged.
- [ ] Verify rendered popups at 375px and desktop widths in both themes: bounds, contrast, chevron clearance, keyboard selection, typeahead, Escape and focus return.
- [ ] Adapt native-only tests to observable house-dropdown use without removing their independent behavior assertions. Record focused checks and full frontend, Admin and authenticated E2E results, actual exit status and any retries; complete the outstanding Standards and Spec reviews before merge.

## Scope

This ticket owns non-Trade controls. Ticket 03 owns Trade controls. Both can use the already installed house Select without blocking each other. Do not add a new dependency or a forwarding-only wrapper, or change Wishlist mutation state or global startup behavior. Report a concrete missing metadata contract rather than inventing one.
