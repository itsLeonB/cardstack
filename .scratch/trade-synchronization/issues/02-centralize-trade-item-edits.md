# 02: Centralize adding and removing Trade Items

**What to build:** Route Add and Remove through the Trade synchronization module so confirmed edits remain visible, old-session replies cannot affect the current view, and the picker returns with the new offer ready. Keep current controls, validation and action-specific errors.

**Blocked by:** 01 — Centralize Trade reads and polling.

**Status:** ready-for-agent

Parent spec: Trade synchronization behind one module.

- [ ] Add and Remove use the existing synchronization module for session-safe write-result handling and required query/router refresh order. Their callers retain layout, validation, action intent, error presentation and navigation destinations without coordinating the two stores themselves.
- [ ] A successful Add adopts the returned Trade, completes required refreshes and returns to the detail page with the addition visible. The caller does not recreate cache-seeding and refresh choreography.
- [ ] Keep selected Card, source Collection, quantity validation, same-Card offers from different Collections and leaving without submitting unchanged. A refused or failed Add keeps its current inline error and does not navigate; a missing source Collection is not treated as a missing Trade.
- [ ] Keep the picker free of Trade polling. If the other participant locks or ends the Trade while the picker is open, the server's submit-time refusal remains the response; do not add automatic redirection or a new lock warning flow.
- [ ] A successful Remove refreshes the offer. Removal remains available only for the viewer's own side while negotiating. HTTP refusal and network failure keep the existing inline removal error and recovery behavior.
- [ ] A delayed Add or Remove reply from an earlier session causes no current-session cache write, message, navigation or refresh, including after account A to B to A. This does not require undoing a write already committed by the server.
- [ ] An earlier poll resolving after a successful edit does not restore the pre-edit offer. Keep ordering decisions inside the module without treating offer revision as complete freshness.
- [ ] Extend existing mounted-route/picker tests with controlled replies for delayed Add and Remove after a session change and one action/poll race. Use the existing production interface; add no test-only export, adapter or flag and do not duplicate the same race in browser tests.
- [ ] Retain the existing authenticated picker round trip, backing-out behavior, validation, refusal, restricted-access and multiple-source-Collection coverage. Test through visible content, inline errors and navigation rather than private state or invalidation call counts.
- [ ] Keep affected cached Trade views and any existing Trades list data changed by these edits current. Use real query keys and a real QueryClient where cache behavior is under test. Do not broaden this into an Inventory Entry, Wishlist or binder refresh redesign.
- [ ] This slice must pass after ticket 01 alone. Do not require a write abstraction introduced by the Join/Accept/Confirm ticket; add only the internal implementation needed by these production callers.
- [ ] Run affected behavioral tests and the repository's compose-backed frontend verification gate, retaining the authenticated picker checks. Report results and any signed-in skips or missing credentials.

## Notes

Keep the current transport contracts and backend Trade rules. Concentrate synchronization knowledge rather than introducing a generic action framework, an invalidation registry or a forwarding module per action.

This ticket and the Join/Accept/Confirm ticket may touch the same module but have no behavioral dependency on each other. Coordinate overlapping edits without inventing a blocking edge. Do not change Create, Cancel, Discard, list presentation or page design.
