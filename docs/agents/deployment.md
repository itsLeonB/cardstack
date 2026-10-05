# Deployment and production

Where cardstack runs, which env vars each deploy needs, and how to find out why a deploy is not showing your change. One-time secret setup is in `README.md`.

## Topology

| Piece | Host |
| --- | --- |
| Frontend (`frontend/`) | Vercel |
| Backend API (`backend/`) | Railway, single `api` service in the `cardstack` project |
| Database | Neon, `production` branch |

Get the live origins from the Vercel and Railway dashboards or CLIs, not from this doc.

## Env vars

Vite bakes `VITE_*` vars into the bundle at build time, so a changed value needs a new build.

- `VITE_SITE_URL`: absolute origin of the frontend, used for the sitemap, `robots.txt` and social-preview URLs. Set it in the Vercel production environment. When it is unset, `bun run build` exits non-zero if `VERCEL_ENV=production` (`frontend/tools/seo-files.ts`), and warns and continues elsewhere.
- `VITE_CLERK_PUBLISHABLE_KEY`: the Clerk instance's publishable key (public; `pk_test_...` or `pk_live_...`). The frontend cannot sign anyone in without it: when it is unset the app renders a "not configured" error instead of running as a guest. `bun run build` exits non-zero if `VERCEL_ENV=production` and it is unset (`frontend/tools/clerk-key.ts`), and warns and continues elsewhere, so local, CI and preview builds without it still pass. Set it in Vercel for Production only, with the production instance's key, which needs a domain the owner controls (a `*.vercel.app` address will not work). Preview deployments, CI and local development use the development instance: the preview workflow sets the branch-scoped value from the GitHub variable `VITE_CLERK_PUBLISHABLE_KEY` (also read by the frontend CI and end-to-end workflows), and `scripts/clerk-setup.sh` writes the local value to `frontend/.env`. Local value: `frontend/.env.example`.
- `VITE_API_BASE_URL`: the API origin. When it is unset the generated client falls back to `http://localhost:8080`, so a production build without it calls localhost. Local value: `frontend/.env.example`.

## Backend sign-in (Clerk)

The API authenticates only by an `Authorization: Bearer` Clerk session token (ADR-0015); it holds no credentials, sessions or cookies. It verifies the signature against the instance's keys (fetched with the secret key and cached), the expiry, the issuer, and that the token's authorized party is one of the frontend origins. The API refuses to boot without the secret key and issuer (the migration job and the ingesters do not need them), and rejects every token (logging why) while no frontend origin is configured. Local values: `backend/.env.example`.

- `CLERK_SECRET_KEY`: the instance's secret key, used to fetch its signing keys. Keep it out of the frontend.
- `CLERK_ISSUER`: the instance's Frontend API URL, the `iss` claim of its tokens (for example `https://example.clerk.accounts.dev`). It is also the base64 payload of the publishable key, which is how the preview and end-to-end workflows derive it.
- `APP_CLIENT_URLS`: the frontend origins, comma-separated, also used for CORS. A token minted for any other origin is rejected with 401. The preview workflow sets it to the Vercel preview URL after deploying it.

Rollout order matters, because nothing enforces it. `preserve()` in `railway.ts` keeps an existing `CLERK_SECRET_KEY` and `CLERK_ISSUER` but never creates them, and the migration runs before the new API starts. Set `CLERK_SECRET_KEY`, `CLERK_ISSUER` and `APP_CLIENT_URLS` (the production frontend origin) in Railway production before merging, or the migration deletes every user and drops the old auth tables while the new API exits at boot and the old one keeps running against the changed schema.

The Clerk session token must carry the `email` and `name` custom claims (`scripts/clerk-setup.sh` adds them). A token without an email claim is rejected.

The first authenticated request creates the user and profile; the migration that introduced this deletes every existing user, with their Collections and Inventory Entries, once, when it first runs on a database.

## Image hosting

Card and Expansion Set images live in a Cloudflare R2 bucket and are served from our own image host (ADR-0016). Three sides read separate settings:

- API (Railway): `IMAGE_BASE_URL`, the public address of the image host with no trailing path (for example `https://img.example.com`). The API serves `IMAGE_BASE_URL` plus a row's hosted key as its image address. With it unset, or on a row with no key, the address is empty. It never falls back to the scraped source address.
- Frontend (Vercel): `VITE_IMAGE_HOST`, the same origin as `IMAGE_BASE_URL`. The frontend rewrites image addresses on that origin to Cloudflare Image Transformations (`/cdn-cgi/image/width=<w>,format=auto,onerror=redirect/<key>`): card tiles at 240 and 480 wide, the card detail view at 720, Expansion Set covers at 128. `onerror=redirect` serves the original when the free transformation allowance is spent. With it unset, or on an address from any other host, the address is used unchanged, so local development needs nothing. It is baked in at build time, and Image Transformations must be enabled on the image host's zone (ticket 02).
- Ingester (`go run ./cmd/ingest-pokemon-asia`, run by hand, never on Railway): `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` and `R2_BUCKET` from an R2 API token scoped to that bucket. `R2_ENDPOINT` is optional and replaces the address derived from the account id. Unless all four of the first group are set, the ingester logs a warning and hosts nothing, so a local ingest of one small set needs no credentials.

Keys are `cards/<card id>` and `expansion-sets/<expansion set id>`, so re-running either command is the backfill: it copies only rows whose key is empty, and a failed download or upload is logged and retried by the next run. To migrate rows that are already in the database without crawling the source site at all, run `make host-images` (`go run ./cmd/host-images`, optionally with `-set <code>`): it reads every Expansion Set and card with a source address and no key, downloads each image under the same rules as the ingester, uploads it to R2, and saves the key. It needs the four `R2_*` settings and exits 1 if any image failed, so re-run it to retry just those. The ingester still hosts images for the rows it ingests, and `go run ./cmd/ingest-pokemon-asia -sync-expansion-sets` hosts Expansion Set covers without crawling any card. Hosted originals are uploaded with `Cache-Control: public, max-age=31536000, immutable`, which is safe because a key always maps to the same bytes.

Rollout order matters, because nothing enforces it. `preserve()` in `railway.ts` keeps an existing `IMAGE_BASE_URL` but never creates one, so set it in Railway first. `make host-images` reads the columns the hosted-images migration adds, so apply the migration before running it: run `make job` against the production database, then `make host-images`, and deploy the API right after the migration. That migration drops `image_url` in the same step, so replicas still running the old API fail on catalog queries from the moment it runs until the new API is live. Until `host-images` has hosted a row, that row's image address is empty and the frontend shows its placeholder.

## Backend edge secret

- `APP_EDGE_SECRET` (Railway production): when set, the API rejects every request without a matching `X-Edge-Secret` header (403, except `/health`) and takes the client address from `CF-Connecting-IP`, ignoring `X-Forwarded-For` and `X-Real-IP`. Configure a Cloudflare Transform Rule on the API's proxied domain that adds `X-Edge-Secret: <value>` to every request.
- Leave it **unset in preview environments** (and locally): they are reached through their own hosting URLs, not Cloudflare, so a set secret would reject every request. When unset, no edge check runs and the client address comes from `X-Real-IP` as before.

## Backend per-user rate limits

Signed-in callers are limited per user (keyed by their verified user id) inside the API, so one account gets one allowance from any number of addresses. Each tier is a token bucket: a caller may spend the burst at once and gets the per-minute number back each minute. The name-searchable card lists (`GET /catalog/cards`, `GET /inventory/cards`, `GET /collections/{id}/entries`) share a tighter search tier, and the three facets routes (`GET /catalog/facets`, `GET /inventory/cards/facets`, `GET /collections/{id}/facets`) share a tighter facets tier; neither spends the general one. Over a limit, the API answers 429 with a `Retry-After` header in whole seconds and a fixed body. Guests are not limited per user: the Cloudflare edge rule and the in-API per-IP backstop (100 requests a second, burst 200) cover them. The API refuses to boot with any of these set to zero or less.

| Setting | Default |
| --- | --- |
| `RATE_LIMIT_USER_PER_MINUTE` / `RATE_LIMIT_USER_BURST` | 300 / 100 |
| `RATE_LIMIT_SEARCH_PER_MINUTE` / `RATE_LIMIT_SEARCH_BURST` | 60 / 30 |
| `RATE_LIMIT_FACETS_PER_MINUTE` / `RATE_LIMIT_FACETS_BURST` | 30 / 15 |

The buckets live in the API's memory: every deploy or restart resets them, and the limits only hold while the API runs as a single replica. Running more than one replica needs a shared store (for example Redis) before these limits mean anything.

## A failed Vercel build keeps serving the old bundle

If production shows stale behavior (missing images, old UI), suspect a failed Vercel build before suspecting the code: the previous deployment stays live. Check in this order:

1. `gh run list --limit 8`: did the CI for your merge pass?
2. `gh api repos/{owner}/{repo}/deployments --jq '.[0:4][] | [.id,.environment,.sha[0:7],.created_at] | @tsv'`: was your sha deployed?
3. `npx vercel inspect <deployment-id-or-url> --logs`: why the build failed (needs `VERCEL_TOKEN`).
4. `curl -s <api-origin>/catalog/series`: does the API itself return the data? If yes, the bug is in the frontend deploy.

## Preview environments

`.github/workflows/preview-environments.yml` provisions a stack per pull request, and its header comment is the authority on what it creates and deletes:

- A Neon branch `preview/pr-<number>`, forked from `production`.
- Neon credentials and the Clerk development instance's `CLERK_SECRET_KEY` and `CLERK_ISSUER` pushed into the Railway PR environment, which Railway creates and deletes itself. After the Vercel preview deploys, its URL is set as `APP_CLIENT_URLS` on that API.
- A Vercel preview deployment with a branch-scoped `VITE_API_BASE_URL` pointing at that Railway environment and a branch-scoped `VITE_CLERK_PUBLISHABLE_KEY` (the development instance's). Closing the PR deletes the Neon branch, the preview deployments and those env vars.

Provisioning skips fork and Dependabot PRs because they get no repo secrets.
