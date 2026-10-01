# Frontend code conventions

These apply to every frontend change, whether made by the root agent or a subagent. Read `general.md` in this folder as well.

- Extract a component once reuse is real, not in anticipation of it.
- Treat everything in the client bundle as public: render no unsanitized HTML and ship no secrets.
- When you couldn't verify a UI change in a browser, say so instead of claiming success.
- Treat `src/generated/**` (orval) and `src/routeTree.gen.ts` as build outputs. Change the source (the OpenAPI spec, the route files) and regenerate: `bun run codegen` for orval, and the TanStack Start Vite plugin rebuilds the route tree.
- Colocate unit tests beside the source as `*.test.ts(x)`. End-to-end Playwright tests live in `e2e/`.
- `.oxlintrc.json` is the source of truth for lint rules, including the `anti-slop/*` set. `bun run lint` must pass.
- On public routes, read auth state with `useSession()` (no redirect) and gate features on `isAuthenticated`; only `requireAuth` under `_authenticated/` redirects.
- Quantity editing goes through `useQuantityBatch` + `QuantityControl`; new surfaces reuse them rather than re-implementing batching, ordering or decline handling.
- The `cardId` entries filter treats an empty list as "no restriction": disable the query when there are no ids rather than sending an empty list.
