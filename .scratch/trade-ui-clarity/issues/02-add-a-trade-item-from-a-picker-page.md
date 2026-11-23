# 02: Add a Trade Item from a picker page

**What to build:** A participant adding a Card to their side of a Trade gets a page of its own. It uses the catalog search they already know, asks for the source Collection and quantity there, and returns them to the Trade detail page with the new Trade Item visible. The detail page no longer carries an inline add form.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] An add action on the Trade detail page opens a dedicated picker route for that Trade.
- [x] The picker reuses the existing catalog search results and its filters, pagination and loading behaviour; no second search implementation is introduced.
- [x] The source Collection and quantity are entered on the picker page and stored on the Trade Item exactly as the inline form stored them.
- [x] Submitting adds the Trade Item through the existing add operation, returns to the Trade detail page, and shows the new item there without a manual refresh.
- [x] A zero or negative quantity, a Collection the caller does not own, a duplicate Card and Collection pair, or a Trade no longer `negotiating` is refused with a plain message that names the cause.
- [x] The same Card can be offered twice from two different Collections, as the domain requires.
- [x] Browser back from the picker returns to the Trade detail page and adds nothing.
- [x] The inline add form is removed from the detail page, and adding and removing remain limited to the caller's own side while the Trade is `negotiating`.
- [x] The picker shows only what the caller may offer and never exposes another user's holdings; Guest, non-participant and not-found behaviour is unchanged.
- [x] Authenticated end-to-end coverage proves the round trip from detail to picker and back, and keeps the existing access, cancel and execute assertions intact.
- [x] Focused checks and the full frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.

## Verification

Red first: `bunx playwright test e2e/trade-join.spec.ts` on the source before the picker existed ran 12 tests, with the six new picker cases failing and the six existing cases passing, exit 1.

Focused checks on final source: `bunx vitest run src/lib/trade-loader.test.ts src/routes/trades/-slug.test.tsx` → 2 files / 40 tests passed, exit 0. Lint, Prettier check and typecheck also passed with exit 0 before the gate.

Full gate on final source: `scripts/verification/frontend.sh --package frontend` → lint 0 warnings/errors, Prettier check passed, `tsc --noEmit` passed, 91 test files / 903 unit tests passed, build passed. Its E2E step exited 1 with 103 passed and one failure in the existing catalog breakpoint test; no source change followed. `CI=true scripts/verification/frontend.sh --package frontend e2e` then passed 104/104 with exit 0. No retry was used in that clean run.

Retries and fixes before the clean run: the first full gate exited 1 with 101 passed and two failures. One was the existing catalog focus test, which passed unchanged in the final gate. The other was the real-API trade exchange: after the picker returned, the detail route used stale preloaded loader data while the polling query held the new Trade view, so the effect invalidated in a loop until the API answered 429. Fixed by seeding the add response into the Trade query cache and calling `router.invalidate()` before the history push; `bunx playwright test e2e/trade-exchange.spec.ts` then passed with exit 0. A theme-dropdown axe check in `trade-join.spec.ts` also scanned a menu that was still closing; it now waits for the menu to be hidden before the scan, keeping the assertion unchanged.

## Review

Two-axis review via T3 `delegate_task` ran in parallel on the committed diff. Standards found no hard violation of a documented rule after the fixes: the picker's failed Trade and Collections loads now have Retry actions, the session-aware Trade loader is shared by both routes and covered by `trade-loader.test.ts`, the quantity limit is named, and `canAddItem` replaces the vague flag. Spec found no missing behaviour after adding an authenticated case that offers the same Card from two different Collections. The repeated picker selection steps in the E2E suite were folded into one helper. The review's only unremedied point is the scope of the shared loader extraction: it touches the detail route as well as the picker, and its existing tests still pass.

Follow-up nitpicks were verified against the code and applied: the picker draft state and its catalog and Collection queries now live in a child rendered only for an editable Trade and keyed by session and Slug, so they reset and do not run for a locked Trade; the render-side session, missing-Trade and load-error fallbacks now come from one shared gate used by both Trade routes, preserving Retry and the guest sign-in prompt. No nitpick was skipped.
