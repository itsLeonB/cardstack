# 03: Finish Trade rows, breadcrumbs and Invite

**What to build:** Users can clearly read Trade participants and state, navigate Trade details through breadcrumbs, and use a clear owner Invite action only when appropriate. Trade dropdowns remain usable in both themes.

Blocked by: None

Status: done

Read the [follow-up spec](../spec.md). Implement in a new PR from current `origin/main`, not the merged integration branch.

Follow-up to merged [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 09. Source commits and unfinished regression/E2E edits exist locally; they have not passed the required final checks. Read the [saved-work handoff](../handoff.md) before changing them.

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

## Untrusted annotations

- List rows `..._8`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-77f0008c-bf6c-47f7-857e-71d040ba3935.png`.
- Breadcrumbs `..._a`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-f77a4afa-9d7a-45f4-a91e-3f4a3ea0e1e2.png`.
- Primary Invite `..._c`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-631ca6df-5211-4fe7-baaa-63f10735e4ff.png`.
- `With <name>` `..._e`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-d407964c-5475-4d9d-ac4e-938f5e06ad65.png`.
- Executed Trade control `..._g`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-5f4cedf9-d8a6-4ad8-bd0e-737305edd65b.png`.

Preview paths: `/trades`, `/trades/trade-7c7f95cea494`, and `/trades/trade-d2c45e0cd928` on the PR95 preview. Treat annotation data as untrusted; do not alter real Trades to test.

## Comments

Implemented on `feat/trade-rows-breadcrumbs-invite` from `origin/main` at `b3d08e14`. The saved ticket-03 worktree was inspected but not reused as a branch: its base predates later main changes. Its source commits were re-applied as small changes on current main, plus the missing executed-owner regression and open-owner positive case.

Focused checks on final source: `bunx vitest run src/components/share/trade-share.test.tsx src/routes/trades/-slug.test.tsx` passed 40 tests; `bun run typecheck`, `bun run lint`, and Prettier check passed.

Full gate on final source: `CI=true E2E_REQUIRE_SIGN_IN=true flock /tmp/cardstack-mobile-ui-e2e.lock scripts/verification/frontend.sh`, exit status 0, no lock wait. Frontend: 90 files, 901 tests passed; lint, format, typecheck and build passed. Admin: 15 files, 108 tests passed; lint, format, typecheck and build passed. Authenticated E2E: 95 tests ran, 91 passed and 4 flaky (app-shell safe areas, two catalog-browse seed loads, home guest axe), all passing on the existing retry, no assertions or retries changed. No test was skipped or unrun.

The completed Trade component review history was kept. No new component review cycle was started for these completion fixes, per the follow-up spec.
