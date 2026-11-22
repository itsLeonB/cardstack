# 01: Name the Trade, label the sides, retire Invite

**What to build:** A participant opens a Trade and can tell at once which Trade it is, which side is theirs, and whether they can still invite someone. The page title follows the counterparty once they join and reads "New trade" while the seat is empty. The viewer's side reads "You". Invite appears only for the owner of an `open` Trade.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] The detail page title shows the counterparty's display name once someone has joined, and "New trade" while the Trade is `open`; the state stays visible as the page description.
- [x] The viewer's side is labelled "You" whether the viewer holds the owner seat or the participant seat; the other side keeps its display name.
- [x] The empty seat keeps its existing "waiting for someone to join" wording.
- [x] No email address, holdings, private identifier or other participant data appears in the new labels.
- [x] Invite is visible only to the owner while the state is `open`; it is absent in `negotiating`, `locked`, `executed` and `cancelled`, and absent for a participant who is not the owner.
- [x] The existing Invite dialog, link text, Copy and native share behaviour are unchanged; no Collection or Wishlist publication control is added.
- [x] A Guest and a signed-in non-participant still receive the same not-found answer, and Trade pages remain unindexed.
- [x] An authenticated end-to-end case shows visible Trade content before asserting Invite is absent, with an `open`-owner positive case and a joined-owner negative case.
- [x] The new labels and action state are checked on phone and desktop widths, in both themes, with visible focus and keyboard access.
- [x] Focused checks and the full frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.

## Verification

Focused check on final source: `cd frontend && bunx vitest run src/routes/trades/-slug.test.tsx` → 40 passed, exit 0. The session-title test was confirmed red before the fix and green after it.

Full gate on final source before the session-title fix: `scripts/verification/frontend.sh --package frontend` → lint, format, typecheck, 90 test files / 909 tests, build and 97 Playwright tests all passed, exit 0. After the fix, three full runs exited 1 on unrelated guest and catalog E2E flakes; an e2e-only rerun passed 97/97 with exit 0, and no source change followed it.

Retries before that clean full run: the first full run exited 1 with a catalog focus flake and a trade-exchange strict-mode failure, where the new counterparty `h1` shared the Inbox heading's name; the heading assertion was scoped to `level: 1`, and an e2e-only rerun passed 97/97 with exit 0. The next full run exited 1 on a Prettier formatting warning, fixed with `bun run format`. After the code-review fixes, four more full runs exited 1 on unrelated guest and catalog e2e flakes; an e2e-only rerun passed 97/97 with exit 0, and the last full run passed 97/97 with exit 0. No source change followed the clean full run.

## Review

Two-axis review against `origin/main` ran in parallel. The Standards axis flagged the stray `.gitignore` entry and component tests that duplicated the E2E seam; both were removed. It also flagged judgement-call duplication and raw label strings, which were left as-is because two short expressions are smaller than a new abstraction. The Spec axis flagged the missing verification record, weak multi-theme and viewport coverage of the new labels, and a misleading `tradeTitle` fallback; the record was added, the E2E assertions now check both themes and phone width, and the fallback reads "Trade" for the impossible case of a joined Trade with no counterparty name. The `level: 1` scope in `trade-exchange.spec.ts` was kept because the new `h1` otherwise matched two headings. A later PR review found that the document title did not share the body's session-version guard, so `head` now uses the generic "Trade" title until the loaded `sessionVersion` matches the current Clerk version, with a focused test for the switch.
