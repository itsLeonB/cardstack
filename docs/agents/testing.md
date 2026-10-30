# Test environment: the compose stack

One compose stack (`compose.yaml` at the repository root) gives agents and CI the same database, API, apps and tooling. Never install a toolchain or provision a database yourself: bring the stack up, then run everything with the verification scripts or `docker compose exec`. How to write the tests (mocking, real Postgres, repository interfaces) lives in `docs/agents/conventions/backend.md`, under "Testing".

## The stack

| Service | What it is |
| --- | --- |
| `db` | `pgvector/pgvector:pg18`, throwaway data (`down` removes it). The tests require pgvector (ADR-0005, ADR-0018). |
| `migrate`, `api` | The unchanged production backend image (`backend/Dockerfile`): its migrate binary once, then the API. |
| `api-ready` | A sidecar whose health check asks the API from outside, because the API image has no shell. `up --wait` returns once the database is ready, the migration finished and the API answers. |
| `web`, `admin` | The user app and the Admin app from production-style images (`frontend/app.Dockerfile`, `admin/app.Dockerfile`), each served by `tools/serve.ts`, which reads that app's own `vercel.json`. |
| `backend-tools`, `frontend-tools` | Long-lived tooling containers with the repository bind-mounted at `/workspace`, running as your user. `backend-tools` has Go, a C compiler (the race detector needs cgo) and the pinned lint tool. `frontend-tools` is the official Playwright image at the version `frontend/package.json` pins, plus bun, and serves both frontend packages. |
| `seed` | Profile `seed`: loads the end-to-end fixture, `backend/internal/adapters/db/postgres/testdata/e2e_seed.sql`. |

Each worktree gets its own stack: the project name is the directory's name, and no host port is published, so containers reach each other by service name (`db`, `api:8080`, `web:3000`, `admin:3001`) and two worktrees can run at once. `docker compose down` leaves nothing behind but the named package caches (`cardstack-go-cache`, `cardstack-bun-cache`, shared by every worktree).

## Run the checks

`scripts/verification/<component>.sh [step]` is the one entrypoint, for agents and for CI. It brings up what the step needs, runs it in the tooling container and stops at the first failure. With no step it runs them all in order.

| Script | Steps |
| --- | --- |
| `backend.sh` | `build`, `vet`, `format`, `lint`, `test` |
| `frontend.sh [--package frontend\|admin]` | `install`, `lint`, `format`, `typecheck`, `test`, `build` for each package, then `e2e` (user app) |

Only `test` (the backend) and `e2e` need the database; every other step starts just its tooling container. Lint, typecheck and unit tests run against the bind-mounted source, so they never need a rebuild. `e2e` rebuilds the `web` image, because the app runs from a built image with no hot reload: a source edit reaches a running app only through a rebuild. `scripts/bootstrap-worktree/<component>.sh` copies the gitignored env files and starts the tooling container; run it once in a fresh worktree.

The first run on a machine pulls images and takes several minutes; a warm run adds a few seconds of image checks. Calls to the stack by hand: `docker compose exec backend-tools go test ./internal/domain/repository/...`. The `DB_*` values the tests read are already set in `backend-tools`.

If Docker is unavailable, fall back to a Postgres 18 with the `vector` extension reachable through the `DB_*` variables of `backend/.env.example`; the schema relies on PostgreSQL 18's native `uuidv7()`, so an older major version does not work.

## Frontend: driving a headless browser (Playwright) for manual UI verification

In the stack, `frontend-tools` already carries the browser: drive it with `docker compose exec -w /workspace/frontend frontend-tools node script.mjs` against `http://web:3000`. The rest of this section is for a session outside the stack, such as a cloud sandbox with its own browser.

`frontend/package.json` declares `playwright` (plain library, not `@playwright/test`) as a devDependency, pinned to an exact version (no `^`) matching the Chromium build baked into this box's base image. Run `bun install` in `frontend/` and then `import { chromium } from "playwright"` resolves normally — no absolute-path import workaround needed, and no `playwright install` should ever be run (that would try to re-download a browser that's already provided).

This split follows from what the environment actually provisions vs. what a project must declare itself:

- **Environment-level** (baked into the base container image, not `scripts/setup-environment.sh` — grep it, it doesn't mention Playwright): the Chromium *binary* at `PLAYWRIGHT_BROWSERS_PATH=/opt/pw-browsers`, plus a global `playwright` CLI/library under `/opt/node22`. This exists purely so re-downloading a ~300MB browser per project is unnecessary.
- **Project-level** (this repo's job, same as any other npm dependency): the `playwright` *npm package* itself. Global npm installs are not on Node's module resolution path (no `NODE_PATH` is set on this box), so a bare `import { chromium } from "playwright"` from `frontend/` only resolves if `frontend/package.json` declares it and `bun install` has run.

The pinned version must match the pre-installed browser revision, or Playwright will refuse to drive it. Check compatibility with:

```sh
cat /opt/node22/lib/node_modules/playwright/node_modules/playwright-core/browsers.json  # chromium revision this Playwright build expects
ls /opt/pw-browsers                                                                     # chromium revision actually on disk
/opt/node22/bin/playwright --version                                                    # the globally pre-installed Playwright version
```

Pin `frontend/package.json`'s `playwright` devDependency to that same version. `chromium.launch()` (default options) then finds and drives `/opt/pw-browsers/chromium-<revision>/chrome-linux/chrome` directly — no `executablePath` override needed once the installed version matches.


## Frontend: end-to-end authentication (Playwright and Clerk)

The e2e layer signs in through the real Clerk development instance (ADR-0015), never a stub of Clerk or of the API's identity endpoints. Two things in `frontend/e2e/support/clerk-auth.ts` do it, both on `@clerk/testing/playwright`: `clerkSetup()` runs once in `e2e/global-setup.ts` and fetches a Testing Token (it lets the run past Clerk's bot protection), and every signed-in spec calls `signInAsTestUser(page)`, which attaches that token to the page and redeems a one-time sign-in token minted for the test user, so the session is real and the API verifies its bearer token like any other. `sign-in.spec.ts` is the thin proof of the form itself: the happy path through Clerk's real sign-in form (`signInThroughForm`, also used by the two redirect-after-login specs in `auth.spec.ts`) and one key failure, a wrong password. Other specs only need a session, so they skip the form. API stubs (`page.route` on `/collections`, `/inventory`) stay where a spec needs deterministic data: the stubs answer, the real token authenticates.

| Name | Kind | Where it comes from |
| --- | --- | --- |
| `CLERK_SECRET_KEY` | GitHub secret | The development instance's secret key (Clerk dashboard, API keys). Mints the Testing Token and the sign-in token. A production key (`sk_live_`) is refused by `clerkSetup`. |
| `E2E_CLERK_USER_EMAIL` | GitHub secret | The dedicated test user's email, created in the development instance by `scripts/clerk-setup.sh` (a `+clerk_test` address, which never sends mail and accepts the code `424242`). |
| `E2E_CLERK_USER_PASSWORD` | GitHub secret | That user's password, from the same wizard run. Only the form-based specs type it. |
| `VITE_CLERK_PUBLISHABLE_KEY` | GitHub variable | The development instance's publishable key (public). `clerkSetup` derives the instance's Frontend API from it; the build bakes it into the app. `e2e.yml` falls back to the development instance's public key when the variable is absent. |

`scripts/clerk-setup.sh` stores all four in GitHub and writes the local values (it keeps the secret key in `backend/.env`).

**Skip and fail behaviour.** Without all three secrets the tests tagged `@signed-in` (the sign-in specs and every spec that signs in) skip with a reason that names the missing variables, and `global-setup.ts` does not call Clerk. The guest specs run exactly as before. This is what happens on fork and Dependabot pull requests, which GitHub gives no secrets: the E2E job still runs there for the guest specs (it falls back to placeholder Clerk values for the API) and only the sign-in specs skip. To keep a misconfigured repository from going green with no sign-in run, `e2e.yml` sets `E2E_REQUIRE_SIGN_IN=true` on pushes and same-repo pull requests; with it set, missing credentials make the signed-in specs fail naming the missing variables (never values). A wrong secret fails loudly in the global setup.

**Run it locally.** `scripts/verification/frontend.sh e2e` seeds the stack's database, rebuilds the `web` image and runs the suite from `frontend-tools` against `http://web:3000`: no development server. The API reads the Clerk development instance (`CLERK_SECRET_KEY`, `CLERK_ISSUER`) from `backend/.env`, which the stack loads when it exists (placeholders otherwise, so the API still boots). The stack names its own origins in the API's `APP_CLIENT_URLS` (`http://web:3000`) and `ADMIN_CLIENT_URLS` (`http://admin:3001`); the development instance accepts them. Put `VITE_CLERK_PUBLISHABLE_KEY`, `E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD` and `CLERK_SECRET_KEY` in `frontend/.env`: `playwright.config.ts` loads it, and the script bakes the publishable key into the app. Without the secrets the signed-in specs skip. To narrow a run, pass Playwright's own arguments through `docker compose exec -w /workspace/frontend -e PLAYWRIGHT_BASE_URL=http://web:3000 frontend-tools bunx playwright test e2e/home.spec.ts --project=chromium`.

The origin `http://web:3000` is not a secure context, which only `localhost` is; Clerk logs a notice about its cookie suffix and carries on. A spec that needs a secure browser API would need a Chromium flag.

**What needs what.**

- Guest specs run without Clerk credentials, but the app only leaves its loading state once clerk-js loads, so they need the publishable key and a network path to Clerk.
- Seeded backend (`backend/internal/adapters/db/postgres/testdata/e2e_seed.sql`, and the real API): `catalog-browse.spec.ts`, `catalog-search.spec.ts` and `breadcrumbs.spec.ts`. They assert the seed's Series, Expansion Sets and cards. The rarity, category and tag filters in `catalog-search.spec.ts` are signed-in specs, because the API locks them for a Guest.
- Stubbed data, no seed: `catalog-guest-lock.spec.ts` (guest; its stub applies the API's guest lock), `shared-collection.spec.ts` (guest; stubs the shared-page API and reaches `/s/...` by client navigation from `/`, because a direct load runs the route loader server-side where browser stubs do not apply), `catalog-infinite-scroll.spec.ts` (signed in, since a Guest never loads past page 1, except the legacy `?page=` spec), the signed-in suites of `app-shell`, `home`, `not-found`, `auth` and `inventory-infinite-scroll`, and `card-scanning.spec.ts` (signed in, with the match, Collection, catalog and bulk-update calls stubbed). The signed-in ones still need the real Clerk instance, and the API running with the same instance so it accepts the token for the calls the specs do not stub.
- Real Clerk instance, real API, no stub: `sign-in.spec.ts` (it opens Collections through the API) and the account-page step of `seo.spec.ts`.

**No secret in artifacts.** The HTML report is uploaded when CI fails, so the signed-in specs run in their own untraced `chromium-signed-in` project, selected by tagging the describe `SIGNED_IN_TAG` (the sign-in helpers throw in an untagged test), and the form helper types credentials through `evaluate`, never `fill()`. The reasoning is in the header of `frontend/e2e/support/clerk-auth.ts`. The one residual is the test user's email in `clerk.signIn`'s own error text; GitHub masks secret values in the job log.

## Seed data for manual verification

Loading `e2e_seed.sql` (`docker compose run --rm seed`) is the quickest way to get rows. Ticket 04's full ingestion (`go run ./cmd/ingest-pokemon-asia`, no flags) scrapes all four Series and can take hours. When you need real rows to manually sanity-check a catalog feature (not a full pre-merge verification), ingest one small Expansion Set instead:

```
go run ./cmd/ingest-pokemon-asia -set SVAL
```

To backfill listing-derived Expansion Set fields (e.g. the Expansion Set cover image) on existing rows without crawling any cards, use `go run ./cmd/ingest-pokemon-asia -sync-expansion-sets` (optionally with `-set`/`-series`).

With no `R2_*` variables set the ingester logs a warning and skips image hosting, so this needs no credentials; catalog image addresses stay empty. See `docs/agents/deployment/images.md` for the settings.

`SVAL` is a small set (~23 cards) — fast enough for quick iteration. Reserve the full multi-series scrape for when a ticket's acceptance criteria actually require verifying against the complete four-Series dataset.
