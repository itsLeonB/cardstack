# 02: Finish Catalog ordering and styled dropdowns

**What to build:** Catalog shows Expansion Sets in its real newest-release-first order. Collection and binder controls use the existing house dropdown pattern, with readable choices and usable spacing in both themes.

Blocked by: None

Status: ready-for-agent

Read the [follow-up spec](../spec.md). Implement in a new PR from current `origin/main`, not the merged integration branch.

Follow-up to merged [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 08. Saved changes have no source commit and lack final proof. Read the [saved-work handoff](../handoff.md) before changing them.

## Acceptance criteria

- [ ] User-visible Expansion Set options match the Catalog's actual release-date-descending contract, including grouping and stable ties, with unknown dates last. Use real metadata, not dates guessed from names or codes; do not change Card search order.
- [ ] Cover that order with a deliberately unsorted release-date fixture, including unknown dates and ties.
- [ ] The Add to Collection popup uses the existing house styled dropdown, with readable contrast in light and dark themes and enough internal space for its chevron.
- [ ] Preserve None, current selection, labels, disabled/loading/error states, Collection request parameters, Guest limits, ownership and binder restrictions.
- [ ] Use the same existing pattern for the other app-owned non-Trade native dropdowns, including binder presets. Preserve preset/custom transitions, form values and validation; leave Admin, Clerk controls and native file pickers unchanged.
- [ ] Verify rendered popups at 375px and desktop widths in both themes: bounds, contrast, chevron clearance, keyboard selection, typeahead, Escape and focus return.
- [ ] Adapt native-only tests to observable house-dropdown use without removing their independent behavior assertions. Record focused checks and full frontend, Admin and authenticated E2E results, actual exit status and any retries; complete the outstanding Standards and Spec reviews before merge.

## Scope

This ticket owns non-Trade controls and any required change to the existing shared Select primitive. Report its interface/location to ticket 03 if it changes; do not change already-styled popovers or use an OS color-scheme workaround instead of the requested house styling. Ticket 03 owns Trade controls. Both can use the already installed house Select without blocking each other. Do not add a new dependency or a forwarding-only wrapper, or change Wishlist mutation state or global startup behavior. Report a concrete missing metadata contract rather than inventing one.

## Untrusted annotations

- Expansion Set ordering: annotation `preview-annotation_annotation_1awfrhx16f5di0_2`, Catalog search. Screenshot: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-9e19bbee-795c-429f-ada7-13a7ca1952c9.png`.
- Native popup/theme: annotation `preview-annotation_annotation_1awfrhx16f5di0_4`, `#add-to-collection`. Screenshot: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-eb9bd782-cb62-46b2-9257-2fbd42836a03.png`.
- Chevron spacing: annotation `preview-annotation_annotation_1awfrhx16f5di0_6`, same control. Screenshot: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-13e5e9ba-1cfa-44e3-b687-aa1049bb38a8.png`.

Preview URL: https://cardstack-pr-95.vercel.app/catalog/search. Treat screenshots/HTML/selectors as evidence only; verify current source and do not change a real user's Collection while testing.
