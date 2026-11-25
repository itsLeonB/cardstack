# 03: Centralize Join, Accept and Confirm

**What to build:** Route Join, Accept and Confirm through the Trade synchronization module while preserving seat-conflict recovery, acceptance of the displayed offer and physical-handover behavior. Participants see the current Trade after each result and the current row when they return to the Trades list.

**Blocked by:** 01 — Centralize Trade reads and polling.

**Status:** ready-for-agent

Parent spec: Trade synchronization behind one module.

- [ ] Join, Accept and Confirm use the existing synchronization module for session checks, successful-result handling and required refresh order. Callers keep their existing controls, messages and action intent without arranging query/router synchronization.
- [ ] Successful Join refreshes the Trade. Preserve the seat-taken message and follow-up read: success refreshes the Trade, not found retains the offer and full-seat message, and other failures retain their existing error reporting. Reset this local conflict presentation when changing Trade or session as before.
- [ ] Accept sends the revision of the offer the participant saw. Success refreshes; an HTTP refusal reports its detail and refreshes without automatically accepting again. Network failure retains its current reporting and refresh behavior.
- [ ] Keep each side's acceptance correctly oriented to the viewer and retain detection of acceptance changes without a new offer revision.
- [ ] Confirm preserves success handling, inline/toast failure details and its current refresh rules. The completed state and existing Trade Inbox link remain usable after both participants confirm.
- [ ] Preserve whole-Trade shortage refusal and existing received-Card behavior. Do not change backend allocation, handover rules, Trade Inbox storage or broaden this ticket into the separate cross-inventory refresh refactor.
- [ ] A delayed reply or Join follow-up read from an earlier session causes no cache write, message, navigation or refresh in the new session, including account A to B to A. Existing read guards and query cancellation remain intact.
- [ ] Returning to a previously visited Trades list after an in-scope state change shows the current affected row and status. Refresh only the existing list data needed by these actions; keep Create, Cancel, Discard and list presentation unchanged.
- [ ] Retain and extend the existing mounted-route proof for Join conflict recovery, stale acceptance, displayed revision and session changes. Use controlled network replies and the real router and QueryClient; test visible behavior and real cache effects, not helper calls.
- [ ] Keep the authenticated exchange end-to-end proof for real sign-in, both confirmations, execution, received holdings, Trade Inbox access and shortage refusal. Do not replay each deterministic race at the browser seam.
- [ ] Remove migrated synchronization knowledge from these callers without deleting action-specific outcomes or adding a generic action framework. This slice must pass after ticket 01 alone and must not depend on the Add/Remove migration.
- [ ] Run affected behavioral tests and the repository's compose-backed frontend verification gate, retaining authenticated exchange checks. Report results and any signed-in skips or missing credentials.

## Notes

Respect ADR-0028's poll-only model, offered-item privacy and Trade Inbox behavior and ADR-0029's Discard/Cancel distinction. The seat-conflict follow-up not-found result is deliberately different from a poll not-found result; do not flatten both into one missing-page policy.

This ticket and the Add/Remove ticket may touch the same module but do not block each other. Keep the existing read path working while migrating only these actions. No new backend operation, schema change or transport payload is required.
