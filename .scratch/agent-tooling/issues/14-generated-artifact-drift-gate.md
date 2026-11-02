# 14: Fail the gate when generated artifacts drift

**What to build:** a step in each verification script, and therefore in CI, that regenerates the committed build outputs and fails when the tree comes out dirty. Today `internal/mocks/**`, `openapi.json`, `frontend/src/generated/**` and `frontend/src/routeTree.gen.ts` are committed, and nothing checks they still match their source: `backend.sh` runs build, vet, format, lint and test, `frontend.sh` runs install, lint, format, typecheck, test and build, and no workflow regenerates anything. A change that edits an interface, a Huma input or the spec, then forgets one regeneration command, passes every existing check.

**Blocked by:** None (can start immediately).

**Status:** open

- [ ] A backend `drift` step (`scripts/verification/backend.sh drift`) runs `make mocks` and `make genspec`, then fails if `backend/internal/mocks/` or `backend/openapi.json` changed, naming each changed path.
- [ ] A frontend `drift` step (`scripts/verification/frontend.sh [--package frontend|admin] drift`) runs `bun run codegen` after the build that writes the route tree, then fails if `src/generated/**` or `src/routeTree.gen.ts` changed, naming each changed path.
- [ ] Both steps are part of `all`, and the backend and frontend CI workflows run them as their own job, so a PR that forgets a regeneration fails before review.
- [ ] The failure message names the command to run (`make mocks`, `make genspec`, `bun run codegen`) and the changed files, so the fix needs no search.
- [ ] Proof: removing a method from a repository interface without regenerating mocks fails the backend step naming the mock file; adding a query parameter to a handler without regenerating fails the backend step on `openapi.json` and the frontend step on the generated client.
- [ ] `docs/agents/testing.md` lists the new step and its rough cost, and `docs/agents/workflows/implement.md`'s verification section names it among the steps that run.

## Comments

2026-11-02, Wishlist session: `SetQuantity` moved from `WishlistRepository` into the service's transaction; `make wire` was run and `make mocks` was not, so `MockWishlistRepository.SetQuantity` reached the commit. Nothing failed; a human review comment found it. The same class covers a handler change whose `openapi.json` is not regenerated, where the frontend then compiles against stale types.
