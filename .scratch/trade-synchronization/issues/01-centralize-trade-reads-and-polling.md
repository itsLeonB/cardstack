# 01: Centralize Trade reads and polling

**What to build:** Keep the Trade detail page's current loading and ten-second polling behavior while one deep synchronization module owns session-safe reads and query/router refresh order. The picker reuses the same guarded initial read without gaining polling. Participants see the same offer, document title, access rules and recovery states, with less coordination in callers.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

Parent spec: Trade synchronization behind one module.

- [ ] Deepen the existing Trade read module cluster. Keep loader data as the source for displayed content and titles, with Query as the network/cache adapter behind the seam. The detail page no longer reconstructs polling comparisons and query/router refresh order.
- [ ] Preserve client-only loading, unindexed pages, viewer permissions, generic titles for old-session data, missing-Trade behavior, inline initial-load failure and Retry. Eligible viewers of an open link retain their existing read/join access.
- [ ] Hide previous-session loader content immediately and reject late reads, including account A to B to A. Preserve bounded loader retries and existing session-wide cache clearing; no previous-session poll may restore an old offer or title.
- [ ] Preserve participant polling every ten seconds, including its existing terminal-status behavior, and keep non-participants from polling. The picker still performs initial loading only.
- [ ] Detect status, acceptance, confirmation and Trade Item availability changes even when offer revision stays unchanged. A revision-only freshness check does not meet this criterion.
- [ ] Preserve poll failure behavior: not found revalidates the route; other failures leave the last displayed loader view intact. Do not apply this rule to the separate Join seat-conflict follow-up.
- [ ] Retain existing loader, route, router session and HTTP retry tests. Extend the existing mounted-route proof for an in-flight poll across session changes, using controlled network replies and the real router and QueryClient.
- [ ] Keep action callers working unchanged until their migration tickets land. This ticket does not require the Add/Remove or Join/Accept/Confirm migration and must pass on its own.
- [ ] Do not add a parallel state system, new dependency, caller-managed freshness counter, backend version field or test-only production seam. Keep useful internal read protections rather than replacing them with shallow forwarding modules.
- [ ] Run the affected behavioral tests and the repository's compose-backed frontend verification gate. Report results, signed-in test skips and any unavailable credentials; do not claim skipped behavior passed.

## Notes

This is a complete read-and-poll slice, not a new unused module followed by later wiring. Keep current action behavior green while making the existing read seam deeper. Its interface is the test surface; assert visible content, titles, recovery and session isolation, not helper calls or the exact implementation sequence.

Respect ADR-0028's poll-only updates and offered-item privacy and ADR-0029's Discard/Cancel distinction. Backend behavior and storage do not change. The two action tickets depend on this working module but do not depend on each other.
