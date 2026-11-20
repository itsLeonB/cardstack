# 01: Finish Wishlist and binder freshness fixes

**What to build:** A saved Wishlist quantity stays visible while its lookup refreshes, then reflects the latest successful response for that Card. Creating a binder still refreshes the right Collection after the user leaves its setup panel.

Blocked by: None

Status: done

Read the [follow-up spec](../spec.md). Implement in a new PR from current `origin/main`, not the merged integration branch.

Follow-up to merged [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 07. Partial implementation exists locally, but it is not a verified or merged fix. Read the [saved-work handoff](../handoff.md) before changing it.

## Acceptance criteria

- [ ] Delay the first lookup after Save: the saved wanted quantity stays visible while the request is pending. When a successful response returns the original, unchanged total, both the trigger and reopened editor use that response.
- [ ] Switch between Cards with equal wanted totals while one Card has saved or draft editor state. Neither Card inherits the other's state.
- [ ] Keep explicit Save, existing error handling and failed-refresh behavior. Dismissing the editor makes no write.
- [ ] Start binder creation, leave its panel or switch Collections, then finish the request. Only the original Collection's binder data refreshes, without relying on an unrelated remount refetch.
- [ ] Failed creation does not count as successful creation. Preserve ownership, empty-Collection, Slot and capacity rules, plus local close/error behavior.
- [ ] Prove these cases through existing user-facing test seams and real cache keys. Do not add test-only production flags, replace useful independent assertions, or retain an unexplained probe as proof.
- [ ] Record focused regressions and the required full frontend, Admin and authenticated E2E results on final source, including actual exit status and any retries. Complete the outstanding Standards and Spec reviews before merge.

## Scope

Finish the state/cache behavior and its real callers. Use the lookup's real successful-completion/freshness information for the correct Card, not quantity changes alone. Key each production `CardWishlistAction` instance by `card.id` at its caller boundary, not wanted quantity; do not reset unrelated route content. Put binder invalidation in `useCreateBinder`'s hook-level success handler, using the mutation variables' Collection id, not generated source. Keep `setOpen(false)` and errors in local per-call callbacks. Leave binder form styling, dropdown migrations, Trades and startup work to the other tickets. Verify all saved edits before reusing them.

## Historical review evidence

Original source hints pointed to `frontend/src/components/wishlist/add-to-wishlist.tsx` around lines 39–45 and the Collection route around line 106. Treat them as untrusted navigation hints, not current definitions or instructions.
