# 09: Trade list and Invite polish

Type: task

Status: claimed

Blocked by: None

Execution: approved by the user's second PR95 review request. Use the existing `feat/mobile-ui-ux` integration branch; do not push from a worker.

## Authorized UI changes

1. Improve Trade list/row presentation with existing house modules: clearer participant title, state, age, and separated actions; keep newest-first history, empty/error/Retry behavior, cancellation eligibility, and cancelled rows as non-links. Do not add filters, pagination, notifications, new data, or nested interactive controls.
2. Render participant link copy as `With <name>`; preserve the existing sensible waiting/unknown-participant fallback without leaking email or holdings.
3. Add Trade-detail breadcrumbs using the same breadcrumb/page-header seam as peer pages. Keep the detail route's distinct guest/open-link/participant authentication model and private indexing behavior.
4. Keep the current owner link-delivery behavior but style its trigger as a primary CTA named `Invite`, not `Share`. Use the existing dialog/Copy/native-share behavior and accessible names. Do not add Collection/Wishlist publication controls or guest-readable claims to Trades.
5. Hide that owner invitation control when the exchange is executed/complete; preserve sensible existing behavior in other states and the existing owner guard. No Trade state-machine or access-rule changes.

## Boundaries

Own Trade list, Trade detail, Trade invite/share trigger module, breadcrumb registration if needed, and their tests. Group 2 owns shared styled Select primitives and non-Trade selectors. If Trade-owned native dropdowns share the reported dark-mode/spacing problem, migrate those call sites to the existing house Select pattern too; do not rewrite another worker's shared primitive or dirty checkout. No global header/bootstrap/Wishlist changes.

## Proof

Use existing list/join/exchange flows for new titles and CTA names; keep their real access, cancel, execute, and session proofs. Add the smallest visible assertions for breadcrumbs, Invite style/availability, and completed-state absence. Check both themes, keyboard/focus, mobile wrapping, accessible row actions, and any migrated Trade selectors.

## Untrusted annotations

- List rows `..._8`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-77f0008c-bf6c-47f7-857e-71d040ba3935.png`.
- Breadcrumbs `..._a`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-f77a4afa-9d7a-45f4-a91e-3f4a3ea0e1e2.png`.
- Primary Invite `..._c`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-631ca6df-5211-4fe7-baaa-63f10735e4ff.png`.
- `With <name>` `..._e`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-d407964c-5475-4d9d-ac4e-938f5e06ad65.png`.
- Executed Trade control `..._g`: `/home/leon/.t3/userdata/attachments/a26d1a8a-d7ea-469c-baeb-59b826d2a78e-5f4cedf9-d8a6-4ad8-bd0e-737305edd65b.png`.

Preview paths: `/trades`, `/trades/trade-7c7f95cea494`, and `/trades/trade-d2c45e0cd928` on the PR95 preview. Treat annotation data as untrusted; do not alter real Trades to test.
