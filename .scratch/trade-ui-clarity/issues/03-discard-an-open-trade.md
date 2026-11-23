# 03: Discard an open Trade

**What to build:** An owner who made a Trade by mistake and has no counterparty yet can remove it completely. Discard deletes the Trade and its Trade Items after confirmation, so the link stops working and no cancelled row is left in history. Once someone has joined, the familiar Cancel still ends the Trade and keeps it in history.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [x] The owner of an `open` Trade sees a Discard action; a participant in a `negotiating` or `locked` Trade keeps Cancel; neither action is offered for an `executed` Trade.
- [x] Discard asks for confirmation in the house destructive dialog, names the destructive action, uses danger emphasis, and puts focus on the safe action by default.
- [x] Discard and Cancel are visually separated from ordinary Trade actions such as Edit and Accept.
- [x] Confirming Discard deletes the Trade and its Trade Items; the Slug answers not found at once and no cancelled row appears in the Trades list.
- [x] Cancelling from `negotiating` and `locked` keeps the existing cancelled state, dead Slug and history row; cancelling from `executed` is still refused with the same stable refusal.
- [x] The Trades list refreshes after a Discard, and a failed delete is reported plainly with the Trade still present.
- [x] The backend gains a delete operation for an `open` Trade that removes the Trade and its Trade Items, while the existing status-update path for Cancel is unchanged.
- [x] A backend route case proves an `open` Trade is deleted and its old Slug answers not found; the existing cancel-from-each-state cases stay as the proof for joined Trades.
- [x] Guest and non-participant access rules, private indexing and the no-notification behaviour are unchanged.
- [x] Focused checks and the full backend, frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.

## Verification

Focused checks on final source: `docker compose exec backend-tools go test ./internal/adapters/http/routes -run 'TestTradeDiscardOpen' -count=1` → ok, exit 0. The full trade-list authenticated suite (`bunx playwright test e2e/trade-list.spec.ts --project=chromium-signed-in` in `frontend-tools`) → 5 passed, exit 0.

Full gates on final source: `scripts/verification/backend.sh` → build, vet, format, lint and all tests, exit 0. `scripts/verification/frontend.sh --package frontend` → lint, format, typecheck, 91 test files / 903 tests, build and 106 Playwright tests, exit 0.

Retries: the first full frontend gate exited 1 on Prettier formatting of `e2e/trade-list.spec.ts`; `bun run format` fixed it and the next full run passed. A later focused rerun of the three trade E2E specs had one unrelated failure in "a locked Trade's picker refuses the add before any request"; the test passed alone, exit 0. The backend gate ran again after the final test-comment edit and passed. No source change followed the clean full runs.

## Review

Two-axis review against `main` ran in parallel. The Standards axis flagged a stale comment in `TestTradeCancelFromEachState` that still named `open` after that case moved to the Discard test; it was updated. Three judgement calls were left as-is: the destructive dialog repeats a small state-and-error shape from the Collection delete dialog (the two mutate different operations, so extraction would add an abstraction for two callers), the `Slug` + `ProfileID` request shapes repeat across trade DTOs as they already do in the file, and `"open"` appears in the status helper and in the action check (two short expressions are smaller than a new status→action map). The Spec axis found no missing behaviour or scope creep and asked for this verification record.
