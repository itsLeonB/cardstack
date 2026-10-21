# Deployment and production

Where cardstack runs, which env vars each deploy needs, and how to find out why a deploy is not showing your change. One-time secret setup is in `README.md`.

## Topology

| Piece | Host |
| --- | --- |
| Frontend, the user app (`frontend/`) | Vercel |
| Admin app (`admin/`) | Vercel, its own project on a sibling subdomain of the user app's domain (ADR-0024) |
| Backend API (`backend/`) | Railway, single `api` service in the `cardstack` project |
| Database | Neon, `production` branch |

Get the live origins from the Vercel and Railway dashboards or CLIs, not from this doc. The production domain stays out of the repository: the Admin app's hostname is chosen at deploy time and written here as `<admin-origin>`.

The Admin app's Vercel project builds from the `admin` directory (Root Directory `admin`). It has no preview environment and no Playwright suite: `.github/workflows/preview-environments.yml` provisions only the user app, and `admin-ci.yml` runs typecheck, lint, format, tests and build on changes under `admin/`.

## Env vars

Vite bakes `VITE_*` vars into the bundle at build time, so a changed value needs a new build.

- `VITE_SITE_URL`: absolute origin of the frontend, used for the sitemap, `robots.txt` and social-preview URLs. Set it in the Vercel production environment. When it is unset, `bun run build` exits non-zero if `VERCEL_ENV=production` (`frontend/tools/seo-files.ts`), and warns and continues elsewhere.
- `VITE_CLERK_PUBLISHABLE_KEY`: the Clerk instance's publishable key (public; `pk_test_...` or `pk_live_...`). The frontend cannot sign anyone in without it: when it is unset the app renders a "not configured" error instead of running as a guest. `bun run build` exits non-zero if `VERCEL_ENV=production` and it is unset (`frontend/tools/clerk-key.ts`), and warns and continues elsewhere, so local, CI and preview builds without it still pass. Set it in Vercel for Production only, with the production instance's key, which needs a domain the owner controls (a `*.vercel.app` address will not work). Preview deployments, CI and local development use the development instance: the preview workflow sets the branch-scoped value from the GitHub variable `VITE_CLERK_PUBLISHABLE_KEY` (also read by the frontend CI and end-to-end workflows), and `scripts/clerk-setup.sh` writes the local value to `frontend/.env`. Local value: `frontend/.env.example`.
- `VITE_API_BASE_URL`: the API origin. When it is unset the generated client falls back to `http://localhost:8080`, so a production build without it calls localhost. Local value: `frontend/.env.example`.

### Admin app env vars

The Admin app (`admin/`) reads two `VITE_*` vars, both baked in at build time. It builds no sitemap and takes no `VITE_SITE_URL`.

- `VITE_CLERK_PUBLISHABLE_KEY`: the same Clerk instance and key as the user app. `bun run build` exits non-zero if `VERCEL_ENV=production` and it is unset (`admin/tools/clerk-key.ts`), and warns and continues elsewhere. Set it in Vercel for Production only, as for the user app. Local value: `admin/.env.example`.
- `VITE_API_BASE_URL`: the API origin. `bun run build` exits non-zero if `VERCEL_ENV=production` and it is unset (`admin/tools/api-address.ts`), because a production build without it would call localhost. Local value: `admin/.env.example`.

The app answers every response with `X-Robots-Tag: noindex` (`admin/vercel.json`), serves a `robots.txt` that disallows everything, and links to nothing in the user app.

## Backend settings, one file each

- `docs/agents/deployment/clerk.md`: sign-in fails or every request gets a 401, `CLERK_SECRET_KEY`, `CLERK_ISSUER`, `APP_CLIENT_URLS`, `ADMIN_EMAILS` (the catalog admin allowlist), `ADMIN_CLIENT_URLS` (the Admin app's origins and what a 404 on an admin route means), session sharing between the two apps, the rollout order of the Clerk migration.
- `docs/agents/deployment/images.md`: card or Expansion Set images missing or empty, `IMAGE_BASE_URL`, `VITE_IMAGE_HOST`, the R2 settings (also needed by the API for admin uploads), the R2 CORS entries for the user app and the Admin app, `make host-images`.
- `docs/agents/deployment/embeddings.md`: `card_embeddings` empty, `GEMINI_API_KEY`, `EMBEDDING_MODEL`, `MATCH_THRESHOLD`, `MATCH_MARGIN`, scans answering 500, `make embed-catalog`, `make collect-embeddings`, enabling pgvector.
- `docs/agents/deployment/scan-photos.md`: scan feedback records nothing, `POST /scan/photo-uploads` or `POST /scan/feedback` answers 503, `SCAN_PHOTO_R2_*`, the private scan-photo bucket, its credential and its CORS rule.
- `docs/agents/deployment/api-protection.md`: the API answers 403 or 429, `APP_EDGE_SECRET`, the Cloudflare rule, `RATE_LIMIT_*`.

## Catalog admin rollout order

Apply the migrations with `make job` against the production database first, then deploy the API, then the frontend. The migration `20261017000001` replaces `curated` on `cards` and `expansion_sets` with `created_by` and `updated_by` (it backfills every row to `pokemon-asia`, and rows that were curated to `admin`), and the API and the ingesters read those columns, so run an ingester only after the migration. An ingester changes a row only while `updated_by` is its own Source (ADR-0021). `make ingest-tcgcollector` is manual only and never runs on Railway: it is a dry run unless `ARGS="-apply"` is passed, and it creates new Expansion Sets unpublished (publish by SQL, see its report).

## Admin app cutover order

One release moves the Admin from `/admin` in the user app to the Admin app (ADR-0024). The cutover is direct, with no legacy mode, so follow the order. The API will not boot without `ADMIN_CLIENT_URLS` while `ADMIN_EMAILS` is set, and the old `/admin` stops working the moment the new API ships.

1. Create the Admin app's Vercel project (Root Directory `admin`, framework Vite) and give it a hostname that is a subdomain of the production domain the user app is on (`<admin-origin>`). Set `VITE_CLERK_PUBLISHABLE_KEY` and `VITE_API_BASE_URL` for Production. Do not deploy yet.
2. In the Clerk dashboard, check that the production instance accepts the new host wherever the user app's host is listed. A subdomain of the instance's domain needs no satellite domain (`docs/agents/deployment/clerk.md`).
3. Add `<admin-origin>` to the CORS rule of the R2 image bucket, for `PUT` with the `Content-Type` header (`docs/agents/deployment/images.md`). Admin uploads from the new host fail their preflight without it.
4. Set `ADMIN_CLIENT_URLS` in Railway production to `<admin-origin>` (a bare origin: scheme and host, no path, no trailing slash). `ADMIN_EMAILS` and the `R2_*` settings are already there. Keep the user app's origins in `APP_CLIENT_URLS` and out of `ADMIN_CLIENT_URLS`: an origin in both lists makes the API refuse to boot.
5. Deploy the API. From here until step 6 finishes, no Admin can write: the admin routes answer 404 to the user app's tokens, and the Admin app is not live yet.
6. Deploy the Admin app, then open `<admin-origin>` signed in with a listed email and confirm the shell loads. A Guest or an unlisted user sees the ordinary not-found page there.
7. Deploy the user app without `/admin`. Its `/admin` address now answers the ordinary not-found page.

Rollback is a revert of the release: revert, redeploy the API, then the user app. `ADMIN_CLIENT_URLS` may stay set in Railway, because the old API ignores it, and the Admin app project can stay up unused.

Local setup for the Admin app (port, Clerk development instance, origin in `ADMIN_CLIENT_URLS`) is in `docs/agents/environment.md`.

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
