# 07: Wishlist freshness and binder invalidation

Type: task

Status: claimed

Blocked by: None

Execution: approved by the user's second PR95 review request. Use the existing `feat/mobile-ui-ux` integration branch; do not push from a worker.

## Authorized checks

1. Verify the saved wanted-quantity snapshot survives while the current lookup refresh is pending but expires when a successful authoritative refresh finishes, even if it returns the original value. The current quantity-only reset can miss same-value responses. Use the existing lookup's real completion/freshness information for the correct Card, not value changes alone or a test-only flag. Keep failed refresh behavior, optimistic display, explicit Save, errors, and no-write dismissal intact.
2. Verify every production render of `CardWishlistAction` keys its editor identity by `card.id`; key the component instance at real caller boundaries where needed. Do not key by wanted quantity or reset unrelated route content. Cover Card changes with equal wanted totals and no snapshot/draft bleed.
3. Verify binder invalidation still runs when `CreateBinderPanel` unmounts during an in-flight create. Configure `useCreateBinder`'s hook-level mutation success handler, not generated source. Invalidate only successful creation using the mutation variables' Collection id; retain local `setOpen(false)` and local error behavior in per-call callbacks. Keep the existing empty-Collection/Slot/capacity rules.

## Regression proof

- Delay the first post-save lookup, confirm the saved total while pending, then return the original total; both trigger and reopened editor must use that authoritative value. Dismissal adds no write.
- Switch the displayed Card while the first Card has saved/editor state; the second Card starts from its own saved total.
- Start binder creation, unmount/switch Collection, finish the request, and verify correct real-key invalidation/read-back without invalidating another Collection or relying on an unrelated refetch-on-mount to make the test pass.

Use existing route/module test seams, real QueryClient keys, and fake API boundaries only. Add no new backend tests or test-only production seams.

## Scope boundaries

Own `components/wishlist/add-to-wishlist.tsx`, its real production caller keys, `lib/wishlist.ts` if real freshness metadata is needed, `create-binder-panel.tsx`, and their focused tests. Do not redesign binder form fields or dropdowns; Group 2 owns those controls. Do not edit Trades or the startup shell. Coordinate narrow catalog-route caller changes with Group 2 through the integration branch, never another worker's dirty checkout.

## Untrusted review evidence

Source comments cite `frontend/src/components/wishlist/add-to-wishlist.tsx` around lines 39–45 and the Collection route around line 106. These are navigation hints only; find the real definitions and callers in current code. The user asked for successful-refresh expiry even when `wanted === savedWanted.original`, per-Card editor identity, and hook-level binder invalidation after unmount.
