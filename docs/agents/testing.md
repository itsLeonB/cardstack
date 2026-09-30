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


## Getting a local Postgres

Repository tests run against a real Postgres per `docs/adr/0005`, using the `DB_*` env vars from `backend/.env.example` (`localhost:5432`, user/password/db all `cardstack`). Whether you provision this yourself depends on the environment:

- **Cloud/remote agent environment** (no human sitting at this machine, no existing Postgres you'd be stepping on): check for and self-provision one. Look for a pre-installed server first (`pg_lsclusters`, `service postgresql status`) before assuming Docker is available or needed — a cloud sandbox may ship a packaged Postgres with no Docker daemon at all. Start it (`service postgresql start`), then create the `cardstack` role/database if missing, matching `.env.example`'s credentials exactly.
- **Local developer environment**: don't touch their setup. Assume they already have Postgres running per `backend/internal/domain/repository/repository_test_helper_test.go`'s existing Docker-based instructions, and let them provision it themselves if not.

### Version mismatch: PG18's native `uuidv7()`

The schema's bootstrap migration relies on PostgreSQL 18's native `uuidv7()` (see `backend/internal/adapters/db/postgres/migrations/20260918000000_bootstrap.sql`) so no extension is needed on a real PG18 instance. A cloud sandbox's pre-installed Postgres package may only offer an older major version (e.g. 16), but that doesn't mean PG18 is unavailable — try to actually install/provision a real PG18 first, e.g. via the OS package manager: check what's actually in the configured apt/yum repos (`apt-cache policy postgresql-18` or equivalent) before assuming it isn't there, and add the PostgreSQL project's own package repository (PGDG, `apt.postgresql.org`) if the default repos don't carry PG18 and the sandbox's network access allows reaching it. A second installed major version doesn't need to replace the first — just point `DB_PORT`/a fresh cluster at the port tests actually use.

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

`SVAL` is a small set (~23 cards) — fast enough for quick iteration. Reserve the full multi-series scrape for when a ticket's acceptance criteria actually require verifying against the complete four-Series dataset.
