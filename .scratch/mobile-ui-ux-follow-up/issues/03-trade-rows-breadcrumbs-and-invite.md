# 03: Finish Trade rows, breadcrumbs and Invite

**What to build:** Users can clearly read Trade participants and state, navigate Trade details through breadcrumbs, and use a clear owner Invite action only when appropriate. Trade dropdowns remain usable in both themes.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

Follow-up to [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 09. Source commits and unfinished regression/E2E edits exist locally; they have not passed the required final checks. Read the [saved-work handoff](../handoff.md) before changing them.

## Acceptance criteria

- [ ] Trade rows clearly show participant, state and age with separate accessible actions. Preserve newest-first history, empty/error/Retry behavior, cancellation eligibility and cancelled rows as non-links; add no nested interactive controls.
- [ ] Participant links read `With <name>` and retain the existing waiting/unknown-name fallback without exposing email or holdings.
- [ ] Trade details use the house breadcrumb/page-header pattern while preserving Guest, open-link and participant access rules and private indexing.
- [ ] Eligible owners see a primary action named Invite that keeps the existing link dialog, Copy and native-share behavior. Do not add Collection/Wishlist publication controls or change Trade access.
- [ ] An executed Trade hides Invite from its authenticated owner. The regression must show visible executed Trade content before asserting absence, with an open-Trade owner positive case; an unloaded or Guest view is not proof.
- [ ] Trade and Trade Inbox dropdowns use the existing styled pattern. Preserve None, selection, quantities, Move rules and validation, keyboard/focus behavior and theme contrast.
- [ ] Update affected E2E interactions without weakening access, cancel, execute or session assertions. Check mobile/desktop rendering, both themes, keyboard/focus and automated accessibility.
- [ ] Record focused checks and the full frontend, Admin and authenticated E2E results on final source, with actual exit status and any retries. Keep the completed component review history; do not start another review cycle for its small fixes.

## Scope

Finish the saved Trade implementation and its missing proof. Ticket 02 owns non-Trade controls; reuse the existing house Select rather than rewriting it. No new Trade states, filters, pagination, notifications or backend changes.
