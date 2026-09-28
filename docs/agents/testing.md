# Backend test environment setup

How an agent should get a working Postgres for backend tests, and how to get real catalog data quickly for manual verification, depending on where the session is running.

## Getting a local Postgres

Repository tests run against a real Postgres per `docs/adr/0005`, using the `DB_*` env vars from `backend/.env.example` (`localhost:5432`, user/password/db all `cardstack`). Whether you provision this yourself depends on the environment:

- **Cloud/remote agent environment** (no human sitting at this machine, no existing Postgres you'd be stepping on): check for and self-provision one. Look for a pre-installed server first (`pg_lsclusters`, `service postgresql status`) before assuming Docker is available or needed — a cloud sandbox may ship a packaged Postgres with no Docker daemon at all. Start it (`service postgresql start`), then create the `cardstack` role/database if missing, matching `.env.example`'s credentials exactly.
- **Local developer environment**: don't touch their setup. Assume they already have Postgres running per `backend/internal/adapters/repository/repository_test_helper_test.go`'s existing Docker-based instructions, and let them provision it themselves if not.

### Version mismatch: PG18's native `uuidv7()`

The schema's bootstrap migration relies on PostgreSQL 18's native `uuidv7()` (see `backend/internal/adapters/db/postgres/migrations/20260918000000_bootstrap.sql`) so no extension is needed on a real PG18 instance. A cloud sandbox's packaged Postgres may only offer an older major version (e.g. 16) with no PG18 available via its package manager. In that case, add a local-only compatibility shim to that database — never to a committed migration or to a developer's own database — so migrations can run:

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
