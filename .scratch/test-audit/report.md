# Test audit: all components

Status: discovery complete, no deletions made. Baseline recorded on branch `test/nearest-cards-scan-filter` at `3886ffc1` (one backend-only commit over `origin/main`, so every baseline failure below is inherited from `main`).

## 1. Corpus

| Component | Files | Test LOC | Declarations |
| --- | --- | --- | --- |
| backend | 101 | 24,135 | 795 |
| frontend | 85 | 18,599 | 755 |
| e2e | 17 | 3,029 | 117 |
| admin | 15 | 2,557 | 98 |
| repo tools | 1 | 159 | 4 |
| **total** | **219** | **48,479** | **1,769** |

5,981 assertions total. No snapshots. Two `t.Skip` guards, both environment checks in `backend/cmd/presize-images/vips_test.go` (`vips`/`vipsheader` not installed).

## 2. Baseline results

| Component | Command | Result |
| --- | --- | --- |
| backend | `scripts/verification/backend.sh test` | all packages ok, 0 failures |
| frontend unit | `scripts/verification/frontend.sh --package frontend test` | 86/86 files passed, 105 s |
| admin unit | `scripts/verification/frontend.sh --package admin test` | 15/15 files passed, 7 s |
| e2e (first run, no `CI`, 6 workers) | `scripts/verification/frontend.sh e2e` | 78 passed, 6 failed, 1.8 min |
| e2e (CI-faithful, `CI=1`, retries 2, 6 workers) | `CI=1 scripts/verification/frontend.sh e2e` | 83 passed, 1 failed, 1.9 min |
| e2e (serialized, 1 worker, no retries) | `bunx playwright test --workers=1` in `frontend-tools` | 80 passed, 4 failed, 8.0 min |

The three runs disagree about *which* tests fail, so the failures are flaky rather than broken tests. Root cause found, see "6b. The API dies under concurrent load".

## 3. Lanes (production-owner boundaries)

| Lane | Files | LOC | Declarations |
| --- | --- | --- | --- |
| backend/routes | 28 | 8,335 | 194 |
| frontend/components/scan | 7 | 4,453 | 153 |
| backend/service | 13 | 4,450 | 201 |
| backend/ingestion | 12 | 3,989 | 134 |
| frontend/routes | 10 | 3,522 | 115 |
| frontend/lib | 26 | 3,401 | 192 |
| e2e | 17 | 3,029 | 117 |
| backend/repository | 14 | 2,953 | 69 |
| frontend/components/collections | 11 | 2,755 | 98 |
| frontend/components/catalog | 11 | 1,712 | 87 |
| admin/components | 4 | 1,457 | 52 |
| backend/http-other | 9 | 1,179 | 58 |
| backend/cmd | 8 | 1,167 | 39 |
| remaining 15 lanes | 49 | 3,695 | 291 |

## 4. Mechanical junk scan

Every pattern the checklist names that a script can detect came back empty.

- Zero-assertion tests: none. The three files with no testify call (`pokemonasia/testdb_test.go`, `tcgcollector/testdb_test.go`, `tools/cmd/merge-ticket/main_test.go`) all assert with `t.Fatal`/`t.Error`, which is the right form for a CLI harness.
- Copied inventories, source/import/string greps, snapshot dumps, self-comparisons, `expect(true)`: none.
- Test-only production seams: none in frontend or admin. The only exports with no production caller are `fakeBinder`, `stubBinderApi`, `catalogCard`, `visitCatalog`, `pagesAsked` and `wishlistLookups`, all inside `src/test-binder-api.ts` and `src/test-catalog.tsx`, which are documented shared fixture modules (`stubGridLayout()` in `src/test-grid-layout.ts` is referenced by `docs/agents/conventions/frontend.md`).
- 860 mock call-shape assertions across 74 files. Sampling the densest (`backend/internal/domain/service/embedding_service_test.go`, 68 in 392 LOC) shows the `.EXPECT()` lines are mockery arrangement, while the assertions target the returned summary and `mock.MatchedBy` predicates that guard batch size and which cards are recorded. `docs/agents/conventions/backend.md` requires exactly this for `crud.Repository` services. Retain.

## 5. Ledger: candidates found

| Mark | Location | Evidence |
| --- | --- | --- |
| R | `frontend/src/lib/page-cut.test.ts` + `admin/src/lib/page-cut.test.ts` | Byte-identical (114 LOC each). Production is a deliberate copy (`admin/src/lib/page-cut.ts` header, ADR-0024), so each package guards its own copy. Duplication is real but the retention bar protects a package contract. |
| R | `frontend/tools/clerk-key.test.ts` + `admin/tools/clerk-key.test.ts` | Identical 31-line files, identical production. A missing publishable key must fail each app's own build, so both stay. |
| C | `admin/tools/serve.test.ts` (69 LOC) vs `frontend/tools/serve.test.ts` (78 LOC) | Admin copy drops the "sends no X-Robots-Tag" table case and asserts a different `noindex` path set. If the admin app should also prove it does not over-tag, the missing rows belong in the admin table, not a new file. |
| R | `frontend/e2e/catalog-guest-lock.spec.ts` (212 LOC) vs `frontend/src/routes/catalog/-guest-lock.test.tsx` (147 LOC) | Same contract title ("ends the first page with 'Sign in to see more' and never asks for page 2"). The e2e side proves it through a real browser, request log, focus and axe; the unit side proves the component gate. Distinct transport risk, so the authoring gate keeps both. |
| R | 17 duplicate test titles across files | All pairs are different subjects (shared Collection vs shared Wishlist page, `card-holdings` vs `master-inventory`) or the cross-package copies above. No duplicated contract. |

No `D` (delete) candidate survived evidence collection.

## 6. Baseline defects (product, not test)

Six e2e failures on `main`; this branch touches only backend and docs.

1. `e2e/app-shell.spec.ts:200` "no horizontal scroll on /auth/login": `h1` never becomes visible.
2. `e2e/home.spec.ts:132` "landing has no horizontal scroll at phone width".
3. `e2e/catalog-infinite-scroll.spec.ts:494` "a legacy `?page=` link opens the first page of the set".
4. `e2e/auth.spec.ts:34` signed in: return to the protected page after login.
5. `e2e/auth.spec.ts:51` signed in: external redirect target ignored.
## 6b. The API dies under concurrent load, and that is the flake

Every mass failure in this session happened after the `api` container had died. `docker compose ps -a` showed `api | exited | Exited (2)`, and its last log lines were `fatal error: concurrent map iteration and map write` from `net/http.Header.Clone` inside `http.Response.WriteHeader`, called by `sentinel-go`'s timeout middleware writing a 503 while the handler goroutine wrote the same response.

`sentinel-go@v0.3.4 httpserver/middleware_timeout.go` runs the handler in a goroutine with a `timeoutWriter` whose `timedOut`/`wrote` fields are plain bools, and on `ctx.Done()` it sets that flag and calls `WriteError(w, 503, ...)` on the **raw** writer, not the wrapper. The request context is done both at the 10 s `config.Global.Timeout` and on a **client disconnect**, so an abandoned request is enough: two goroutines write one `http.ResponseWriter`, and the runtime map fatal kills the process. It is a runtime fatal, not a panic, so the Huma recovery middleware cannot catch it. v0.3.4 is the latest published version, so no upgrade fixes it.

Evidence:

- Before the restart the same spec set was 8/8 red with `alert: "Failed to fetch"` in the page snapshot; after `docker compose up -d --wait api` the identical command was 8/8 green in 16 s.
- Reproduction: 1000 concurrent `curl -s -m 0.002 -m http://api:8080/catalog/series` from `backend-tools` kills the container with the same `fatal error` (observed twice, once on the first burst, once after three bursts survived). Each burst is about 5 s.
- Worker count does not change the failure set: `--workers=2` and `--workers=6` produced the same 17 of 18 failures, because both ran against a dead API.
- CI is green because 2 workers x 84 tests abandon fewer in-flight requests, not because anything is serialized: `step_e2e` has no lock.

Fix direction: replace `sentinelGin.Timeout` in `backend/internal/adapters/http/setup_sentinel.go` with a local middleware that applies `context.WithTimeout` and never writes its own response, letting the handler's single error path write the 503. Keep the crash as a Go test seam at `internal/adapters/http` (abandoned requests must not kill the process).

6. `e2e/sign-in.spec.ts:42` wrong password: fixed in `frontend/e2e/support/clerk-auth.ts` by bounding the `/prepare_second_factor` wait and giving the Cold Clerk form 15 s; `frontend/e2e/app-shell.spec.ts` "no horizontal scroll" now waits 15 s for its first heading, because a slow Clerk request measured 4.6 s against this suite's 5 s default.

The four flaky declarations from the gate split in two. `catalog-browse.spec.ts:8`, `:36` and `breadcrumbs.spec.ts:41` depend on the API: their captured page snapshot is the app's crash fallback ("Something went wrong"), so they are tickets 01 and 02, and no expect timeout touches them. `app-shell.spec.ts:200` does not touch the API at all: with the API stopped those three mobile scroll tests pass 9/9, and the same assertions pass 18/18 at a 15 s timeout.

## 6a. Parallelism: what the scripts actually constrain

`scripts/verification/lib.sh` has one serialization mechanism, `serialized()` (host-wide `flock`), and only `step_test` uses it: `RUN_DIR=$pkg serialized run bun run test`. `step_e2e` calls `bunx playwright test` bare, and `frontend/playwright.config.ts` sets `fullyParallel: true` with no `workers`, so a suite run takes Playwright's default of half the cores (6 here). `docs/agents/testing.md` says the same: the frontend `test` step is the only step under the host-wide lock. Serializing e2e makes the run 8 minutes instead of 2 and does not remove the failures.

## 7. Documentation gaps found while auditing

- `docs/help/testing/writing-tests.md`, cited by the test-audit skill's authoring gate, does not exist.
- `.agents/skills/openclaw-testing/SKILL.md#proof-policy`, cited by SKILL.md and `references/audit.md`, does not exist in this repo.
- `references/audit.md` validation steps name `pnpm deadcode:*`, `node scripts/check-changed.mjs` and `scripts/check-conventions.sh`-style gates that this repo does not have (it uses `scripts/verification/<component>.sh`).
- No component `AGENTS.md` exists under `backend/`, `frontend/` or `admin/`, which CAMPAIGN.md step 5 asks each campaign to update.

The skill was ported from a pnpm monorepo; its cross-references need repointing at this repo before campaign lanes rely on them.

## 8. Recommendation

The mechanical pass found no junk, and the sampled deep reads found disciplined behavior proof. Expected deletion yield of a full 219-file campaign is low. Spend the campaign budget on the two lanes where layer redundancy is plausible instead of all 30: `backend/routes` (8,335 LOC, 194 declarations, four layers per feature: handler, routes, service, repository) and `frontend/components/scan` (4,453 LOC, 153 declarations, which also has `page-scan.test.tsx` and `scan-screen.test.tsx` covering separate screens of one flow).
