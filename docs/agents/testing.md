# Backend test environment setup

How an agent should get a working Postgres for backend tests, and how to get real catalog data quickly for manual verification, depending on where the session is running. How to write the tests (mocking, real Postgres, repository interfaces) lives in `docs/agents/conventions/backend.md`, under "Testing".

## Frontend: driving a headless browser (Playwright) for manual UI verification

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
| `VITE_CLERK_PUBLISHABLE_KEY` | GitHub variable | The development instance's publishable key (public). `clerkSetup` derives the instance's Frontend API from it; the dev server bakes it into the app. `e2e.yml` falls back to the development instance's public key when the variable is absent. |

`scripts/clerk-setup.sh` stores all four in GitHub and writes the local values (it keeps the secret key in `backend/.env`).

**Skip and fail behaviour.** Without all three secrets the tests tagged `@signed-in` (the sign-in specs and every spec that signs in) skip with a reason that names the missing variables, and `global-setup.ts` does not call Clerk. The guest specs run exactly as before. This is what happens on fork and Dependabot pull requests, which GitHub gives no secrets: the E2E job still runs there for the guest specs (it falls back to placeholder Clerk values for the API) and only the sign-in specs skip. To keep a misconfigured repository from going green with no sign-in run, `e2e.yml` sets `E2E_REQUIRE_SIGN_IN=true` on pushes and same-repo pull requests; with it set, missing credentials make the signed-in specs fail naming the missing variables (never values). A wrong secret fails loudly in the global setup.

**Run it locally.** Run the backend against a seeded Postgres as for any e2e run, and give the backend the same development instance (`CLERK_SECRET_KEY`, `CLERK_ISSUER` from `backend/.env.example`; `APP_CLIENT_URLS` must include `http://localhost:3000`). Then, in `frontend/`, put `VITE_CLERK_PUBLISHABLE_KEY`, `E2E_CLERK_USER_EMAIL`, `E2E_CLERK_USER_PASSWORD` and `CLERK_SECRET_KEY` in `.env` (`playwright.config.ts` loads it; real environment variables win) and run `bunx playwright test`. `--project=chromium-signed-in` runs only the signed-in specs, `--project=chromium` only the guest ones.

**What needs what.**

- Guest specs run without Clerk credentials, but the app only leaves its loading state once clerk-js loads, so they need the publishable key and a network path to Clerk.
- Seeded backend (`backend/internal/adapters/db/postgres/testdata/e2e_seed.sql`, and the real API): `catalog-browse.spec.ts`, `catalog-search.spec.ts` and `breadcrumbs.spec.ts`. They assert the seed's Series, Expansion Sets and cards. The rarity, category and tag filters in `catalog-search.spec.ts` are signed-in specs, because the API locks them for a Guest.
- Stubbed data, no seed: `catalog-guest-lock.spec.ts` (guest; its stub applies the API's guest lock), `catalog-infinite-scroll.spec.ts` (signed in, since a Guest never loads past page 1, except the legacy `?page=` spec), the signed-in suites of `app-shell`, `home`, `not-found`, `auth` and `inventory-infinite-scroll`, and `card-scanning.spec.ts` (signed in, with the match, Collection, catalog and bulk-update calls stubbed; it needs `VITE_SCAN_ENABLED=true`, which `playwright.config.ts` sets for the dev server and the preview workflow sets for previews). The signed-in ones still need the real Clerk instance, and the API running with the same instance so it accepts the token for the calls the specs do not stub.
- Real Clerk instance, real API, no stub: `sign-in.spec.ts` (it opens Collections through the API) and the account-page step of `seo.spec.ts`.

**No secret in artifacts.** The HTML report is uploaded when CI fails, so the signed-in specs run in their own untraced `chromium-signed-in` project, selected by tagging the describe `SIGNED_IN_TAG` (the sign-in helpers throw in an untagged test), and the form helper types credentials through `evaluate`, never `fill()`. The reasoning is in the header of `frontend/e2e/support/clerk-auth.ts`. The one residual is the test user's email in `clerk.signIn`'s own error text; GitHub masks secret values in the job log.

## Getting a local Postgres

Repository tests run against a real Postgres per `docs/adr/0005-backend-tests-hit-real-postgres.md`, using the `DB_*` env vars from `backend/.env.example` (`localhost:5432`, user/password/db all `cardstack`). Whether you provision this yourself depends on the environment:

- **Cloud/remote agent environment** (no human sitting at this machine, no existing Postgres you'd be stepping on): check for and self-provision one. Look for a pre-installed server first (`pg_lsclusters`, `service postgresql status`) before assuming Docker is available or needed — a cloud sandbox may ship a packaged Postgres with no Docker daemon at all. Start it (`service postgresql start`), then create the `cardstack` role/database if missing, matching `.env.example`'s credentials exactly.
- **Local developer environment**: don't touch their setup. Assume they already have Postgres running per `backend/internal/domain/repository/repository_test_helper_test.go`'s existing Docker-based instructions, and let them provision it themselves if not.

### Version mismatch: PG18's native `uuidv7()`

The schema's bootstrap migration relies on PostgreSQL 18's native `uuidv7()` (see `backend/internal/adapters/db/postgres/migrations/20260918000000_bootstrap.sql`) so no extension is needed for that on a real PG18 instance. The card embeddings migration does need one: `CREATE EXTENSION vector` (pgvector), so the Postgres under the tests must ship it (the `pgvector/pgvector:pg18` image, or `postgresql-18-pgvector` from the PGDG repository below). The compatibility shim further down covers only `uuidv7()`, not pgvector. A cloud sandbox's pre-installed Postgres package may only offer an older major version (e.g. 16), but that doesn't mean PG18 is unavailable — try to actually install/provision a real PG18 first, e.g. via the OS package manager: check what's actually in the configured apt/yum repos (`apt-cache policy postgresql-18` or equivalent) before assuming it isn't there, and add the PostgreSQL project's own package repository (PGDG, `apt.postgresql.org`) if the default repos don't carry PG18 and the sandbox's network access allows reaching it. A second installed major version doesn't need to replace the first — just point `DB_PORT`/a fresh cluster at the port tests actually use.

Only when a genuine PG18 truly cannot be provisioned in that environment (no compatible package available, no network access to fetch one) fall back to a local-only compatibility shim on the older-version database — never on a committed migration or on a developer's own database — so migrations can still run:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid AS $$
DECLARE
  unix_ts_ms bytea;
  uuid_bytes bytea;
BEGIN
  unix_ts_ms := substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3);
  uuid_bytes := unix_ts_ms || gen_random_bytes(10);
  uuid_bytes := set_byte(uuid_bytes, 6, (b'0111' || get_byte(uuid_bytes, 6)::bit(4))::bit(8)::int);
  uuid_bytes := set_byte(uuid_bytes, 8, (b'10' || get_byte(uuid_bytes, 8)::bit(6))::bit(8)::int);
  RETURN encode(uuid_bytes, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;
```

This is purely to unblock `goose` migrations and tests in a sandbox that can't run real PG18 — it doesn't change what ships. No test asserts actual v7 bit-layout, only that primary keys are non-nil valid UUIDs.

## Seed data for manual verification

Ticket 04's full ingestion (`go run ./cmd/ingest-pokemon-asia`, no flags) scrapes all four Series and can take hours. When you need real rows to manually sanity-check a catalog feature (not a full pre-merge verification), ingest one small Expansion Set instead:

```
go run ./cmd/ingest-pokemon-asia -set SVAL
```

To backfill listing-derived Expansion Set fields (e.g. the Expansion Set cover image) on existing rows without crawling any cards, use `go run ./cmd/ingest-pokemon-asia -sync-expansion-sets` (optionally with `-set`/`-series`).

With no `R2_*` variables set the ingester logs a warning and skips image hosting, so this needs no credentials; catalog image addresses stay empty. See `docs/agents/deployment/images.md` for the settings.

`SVAL` is a small set (~23 cards) — fast enough for quick iteration. Reserve the full multi-series scrape for when a ticket's acceptance criteria actually require verifying against the complete four-Series dataset.
