# 05: Hosted images in the ingester and API

**Parent:** `.scratch/security-hardening/spec.md` (see ADR-0016)

**What to build:** Running the pokemonasia ingester copies each card image and each Expansion Set cover into our R2 bucket, and the API serves hosted addresses. After a run, a catalog response's image address points at our image host instead of the source site; re-running only copies what is missing.

**Blocked by:** None (can start immediately). Real-bucket verification needs ticket 02; the code and tests use a fake object store.

**Status:** ready-for-agent

- [ ] Cards and Expansion Sets gain a hosted-image key column; the scraped source address moves to its own column. The migration backfills the source column from the current address and leaves the key empty.
- [ ] The ingester downloads each image from the known source host and uploads the unmodified original to R2 under a deterministic key per row (one namespace for cards, one for Expansion Sets), through a small object-store interface with an R2 adapter.
- [ ] Re-running skips keys already hosted, and the CLI's existing flags (single set, sync-expansion-sets) keep working, including for backfilling Expansion Set covers without crawling cards.
- [ ] Only the known source host is fetched; a maximum size is enforced; content that is not an image is rejected before upload.
- [ ] A failed download or upload leaves the key empty, is logged, and does not abort the run; a later run retries it.
- [ ] With R2 not configured the ingester logs a warning and skips hosting, so a local ingest of one small set needs no credentials.
- [ ] The API's card and Expansion Set image address is the configured image base address plus the key, and is empty when there is no key. It never falls back to the source address.
- [ ] Ingester tests (fake store, local test server for the source, real Postgres) cover: upload and key assignment, skip-if-hosted, failure without abort, host allow-list, size and content checks, no-R2 path. Catalog route tests cover the new address field.
- [ ] The OpenAPI file is regenerated if the contract changed, the new settings are documented in the deployment doc, and backend build, vet and tests pass.

## Comments
