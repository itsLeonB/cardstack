# Trade synchronization behind one module

Status: ready-for-agent

## Problem Statement

A Trade participant needs the detail page to show the current offer after joining, adding or removing a Trade Item, accepting the offer, or confirming the handover. The page must also show changes made by the other participant without bringing back old data after an account change. Today, the detail page, picker and action callers each arrange parts of the session checks, cache updates and route refreshes. This split makes a future change risky: a caller must know how two stores work and which refresh must finish first. Code exploration confirmed this repeated knowledge; it did not establish a user-visible stale-data bug.

## Solution

Keep the current Trade experience while giving one deep Trade synchronization module ownership of session-safe reads, polling, successful-write updates and refresh order. Participants keep the same titles, controls, errors, navigation and ten-second detail polling. The picker keeps submit-time refusal rather than gaining polling. Callers use the module's interface without coordinating the query cache and router themselves.

## User Stories

1. As a Trade participant, I want the detail page to load the Trade I can access, so that I can review the offer before acting.
2. As a Trade participant, I want the page title and displayed offer to belong to my current session, so that switching accounts does not reveal the previous account's Trade.
3. As a user switching accounts, I want old Trade content hidden at once, so that revalidation does not expose it while the new read is pending.
4. As a user switching from one account to another and back, I want the first session's delayed responses rejected, so that signing into the same account again does not revive an old request.
5. As a Trade participant, I want failed initial loading to keep its Retry action, so that I can recover without leaving the page.
6. As a person opening an unavailable Trade, I want the current not-found and sign-in behavior preserved, so that this refactor does not change who can discover a Trade.
7. As a Trade participant, I want the detail page checked every ten seconds, so that I can see changes from the other participant.
8. As a participant viewing an unchanged offer revision, I want changes in acceptance, confirmation and Card availability still detected, so that the page does not mistake the offer revision for a complete freshness marker.
9. As a participant whose Trade disappears during polling, I want the existing missing-Trade behavior, so that I do not keep acting on a Trade that no longer exists.
10. As a participant whose poll fails temporarily, I want the last displayed offer retained as it is today, so that a failed refresh does not replace the page with a new error flow.
11. As an eligible signed-in person opening an open Trade link, I want to join through the existing action, so that I can take the second seat.
12. As a person whose join loses to another person, I want the existing full-seat message, so that I understand why joining failed.
13. As a person whose seat-conflict follow-up read returns not found, I want the offer and full-seat message retained, so that this expected conflict does not become a generic missing-page result.
14. As a person whose seat-conflict follow-up read succeeds, I want the Trade refreshed, so that an already-joined state can recover.
15. As a participant, I want to add a Card with a quantity and source Collection through the existing picker, so that the offer still records the intended copies.
16. As a participant who adds a Trade Item successfully, I want to return to the detail page with that addition visible, so that I can check the resulting offer immediately.
17. As a participant whose addition fails, I want the current inline error and picker state retained, so that I can correct the input or decide what to do next.
18. As a participant who leaves the picker without submitting, I want no Trade Item added, so that browsing Cards alone does not change the offer.
19. As a participant whose Trade locks or ends while the picker is open, I want the existing submit-time refusal, so that the refactor does not add a new polling or navigation rule.
20. As a participant, I want to remove my own Trade Item while negotiating, so that I can change my offer without changing the other participant's side.
21. As a participant whose removal fails, I want the existing inline error, so that I know the item was not confirmed as removed.
22. As a participant accepting a deal, I want acceptance tied to the offer revision I saw, so that I do not silently accept a different offer.
23. As a participant whose acceptance is stale, I want the existing error and refreshed offer without automatic re-acceptance, so that I can review the changed deal myself.
24. As a participant, I want my accepted state and the other participant's accepted state displayed correctly, so that I know who still needs to act.
25. As a participant confirming a physical handover, I want the existing confirmation result and refresh, so that the exchange still completes only under the current Trade rules.
26. As a participant whose execution fails because a Card is short, I want the existing error detail and unchanged inventory behavior, so that no partial exchange occurs.
27. As a participant whose exchange completes, I want the existing completed state and Trade Inbox link, so that I can find the Cards I received.
28. As a user changing sessions during an action, I want the old reply to cause no cache write, message, navigation or refresh in the new session, so that the previous session cannot affect my current page.
29. As a participant receiving a delayed poll after a successful action, I want the displayed Trade not to revert to the earlier offer, so that completed actions remain visible.
30. As a participant returning to a previously visited Trades list, I want the affected row to show the current Trade state, so that I can find the correct next action.
31. As a non-participant allowed to view an open Trade, I want the existing read and join permissions without participant polling, so that access does not expand through this refactor.
32. As a participant, I want Trade pages to remain unindexed and limited to offered Trade Items, so that synchronization does not expose private holdings or contact details.
33. As a maintainer, I want session checks and refresh ordering concentrated in one module, so that changing an action does not require rebuilding that knowledge in its caller.
34. As a maintainer, I want existing behavioral tests to survive the refactor, so that module depth improves without weakening proof of the Trade experience.

## Implementation Decisions

- Scope covers the Trade detail page, the add-item picker, and Join, Add, Remove, Accept and Confirm. Create, Cancel and Discard from the Trades list remain unchanged. Include refresh of the existing Trades list only where an in-scope action needs it; do not redesign that list.
- Deepen the existing Trade read module cluster rather than creating a parallel state system. Its interface hides session-safe reads, detail polling, adoption of successful write results and the required query/router refresh order. Depth must remove coordination knowledge from callers, not merely move comparison code into a shallow module.
- Keep loader data as the source for the displayed Trade and document title. Keep Query as the network/cache adapter behind the synchronization seam. Do not migrate the page to rendering directly from Query or remove the router loader during this work.
- Keep page layout, labels, action intent and navigation destinations with their callers. The module owns when a current-session result is safe to adopt and when required refreshes have completed; callers must not reconstruct cache-seeding and invalidation sequences.
- Keep action-specific outcomes distinct. The module's implementation may use internal helpers, but do not introduce a configurable action framework, a generic invalidation registry, a pass-through module per action, or a new dependency.
- Preserve session-version checks, immediate hiding of old loader data, the generic title while data belongs to an older session, bounded read retries, and existing session-wide cache clearing. Reject late responses even across an account A to B to A change. Old responses must not write cache data, show messages, navigate, or refresh the new session's view.
- Preserve client-only Trade loading, the existing document-title rules and unindexed pages. Keep the established Guest, eligible open-link viewer and participant access rules; do not replace them with a blanket non-participant denial.
- Keep participant detail polling at ten-second intervals and keep its current status eligibility, including terminal statuses. Keep non-participants from polling. The picker does not gain polling and continues to rely on server refusal if editability changes before submission.
- Preserve observation of acceptance, confirmation, status and Trade Item availability independently of offer revision. Offer revision describes changes to the offer, not every change to the Trade view. Do not replace the existing checks with revision-only freshness.
- A delayed read must not restore a view made obsolete by a successful action. Coordinate reads and writes inside the module using the existing query and session facilities; do not add a persisted sequence, backend version field or caller-managed freshness counter.
- Preserve initial-load missing and failure states, their current sign-in and Retry behavior, and the existing handling of poll failures. A poll not-found result revalidates the route; another failed poll retains the last displayed loader view.
- Preserve Join's seat-conflict recovery. A successful follow-up read refreshes the Trade; a not-found follow-up keeps the offer and full-seat message; other follow-up failures keep their current error handling. Do not make every not-found result clear the page.
- Preserve Add validation, the selected Card, quantity and source Collection rules, and inline failure reporting. A successful addition adopts the returned Trade, completes the required refresh, and returns to the detail page. A refused addition does not navigate. A source Collection not-found response is not proof that the Trade is missing.
- Preserve Remove authorization, negotiation-only controls and inline failure reporting. Preserve Accept's displayed offer revision, error reporting and refresh on HTTP refusal, with no automatic re-acceptance. Preserve Confirm's current inline/toast errors and refresh behavior. Do not collapse these into one uniform refusal policy.
- Keep the existing transport contracts, payloads, backend Trade rules and storage unchanged. No schema migration, generated contract change or new backend operation is needed.
- Respect ADR-0028's poll-only updates, offered-item privacy, whole-Trade shortage refusal and Trade Inbox behavior. Respect ADR-0029's separate Discard and Cancel rules. This refactor does not reopen either decision.

## Testing Decisions

- Use the agreed module interface as the test surface. Prefer the highest existing observable seam: mounted Trade routes with the real router and QueryClient and controlled network replies. Drive actions and assert visible content, messages, titles and navigation, not private state, helper calls, request choreography or the number of internal modules.
- The primary deterministic proof belongs to the existing Trade detail route tests. Retain their coverage of session changes, A to B to A, hidden old content, titles, joining, seat-conflict recovery, removal, displayed acceptance revision, stale acceptance, availability changes without revision changes, poll not-found, non-participant non-polling and locked controls.
- Retain the existing loader, router session and HTTP retry tests as owners of their distinct lifecycle contracts: bounded retries, session-wide cache clearing, loader revalidation and preventing an authenticated retry from crossing sessions. Do not replay each contract in every action suite.
- Extend the route-level proof with delayed action replies after a session change, including Add and Remove, and with an in-flight poll across A to B to A. Assert that old content, messages, navigation and cached Trade results do not return. Use existing fixtures and controlled promises; do not expose production controls used only by tests.
- Add one focused race case in which an earlier poll resolves after a successful action. Assert that the current offer remains displayed. Do not use offer revision alone in the fixture to make acceptance or availability changes disappear.
- Keep existing picker coverage for add-and-return, leaving without adding, quantity validation, refusal, restricted access and offering the same Card from two Collections. Use an existing picker or route test setup for deterministic delayed-reply coverage rather than introducing another public seam.
- Retain the authenticated Trade exchange end-to-end flow as distinct proof of real Clerk, transport, persistence, both confirmations, Trade Inbox access and whole-Trade shortage refusal. Keep its received-Card assertions. Do not duplicate every controlled race in browser tests.
- Prove that an affected row is current when returning to a previously visited Trades list after an in-scope state change. Use a real QueryClient for cache behavior; a mocked invalidation function cannot prove that real query keys refresh the intended data.
- Preserve action-specific failure behavior in existing cases, especially seat-conflict plus follow-up not-found, Add's source Collection refusal, Remove's inline error, and Accept/Confirm resynchronization. Extend a case only where the refactor introduces a distinct lifecycle risk.
- A good test fails on a credible user-visible regression and survives renaming or moving the implementation. Give each behavior one primary owner. Add no test-only exports, flags, adapters or injection seams, and do not delete useful tests because the implementation moved.
- Verification for implementation uses the repository's compose-backed frontend verification entrypoint for lint, formatting, type checking, tests, build and authenticated end-to-end tests. Report command results and any missing credentials or skipped signed-in cases. Spec publication alone does not establish that implementation or runtime behavior passes.

## Out of Scope

- No new polling in the picker, changed detail poll interval, push updates, notifications, automatic acceptance or new terminal-status polling rule.
- No changes to Trade state transitions, authorization, offered-item privacy, source-first allocation, shortage refusal, physical handover rules or Trade Inbox storage.
- No changes to Create, Cancel or Discard behavior, their dialogs, or Trades list presentation.
- No broad Inventory Entry, Master Inventory, Wishlist or binder refresh refactor. That was a separate architecture candidate and is not part of this work.
- No redesign of Trade cards, labels, controls, catalog search, picker forms, routing destinations or error wording.
- No backend refactor, schema change, new transport contract, new state-management dependency or new hypothetical adapter.

## Further Notes

The user confirmed the scope, retained picker submit-time refusal, and the ownership decision before requesting this spec. The test-seam discussion also confirmed behavior-level proof through the module's interface. No further interview or seam approval is needed for publication.

The design aims for locality of session and refresh rules and leverage across the detail page, picker and actions. Apply the deletion test: keep the existing read and session protections that would otherwise spread into callers, and remove duplicated coordination rather than adding another shallow module over it.

Code exploration established differing local protections for mutation and poll replies, not a confirmed cross-session leak or stale-data failure. Implementation must use targeted tests to distinguish existing protection from gaps and report any newly proven defect. No implementation, browser run or test run took place during specification.
