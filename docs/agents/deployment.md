# Deployment and production

Where cardstack runs, which env vars each deploy needs, and how to find out why a deploy is not showing your change. One-time secret setup is in `README.md`.

## Topology

| Piece | Host |
| --- | --- |
| Frontend (`frontend/`) | Vercel |
| Backend API (`backend/`) | Railway, single `cardstack` service |
| Database | Neon, `production` branch |

Get the live origins from the Vercel and Railway dashboards or CLIs, not from this doc.

## Env vars

Vite bakes `VITE_*` vars into the bundle at build time, so a changed value needs a new build.

- `VITE_SITE_URL`: absolute origin of the frontend, used for the sitemap, `robots.txt` and social-preview URLs. Set it in the Vercel production environment. When it is unset, `bun run build` exits non-zero if `VERCEL_ENV=production` (`frontend/tools/seo-files.ts`), and warns and continues elsewhere.
- `VITE_API_BASE_URL`: the API origin. When it is unset the generated client falls back to `http://localhost:8080`, so a production build without it calls localhost. Local value: `frontend/.env.example`.

## Image hosting

Card and Expansion Set images live in a Cloudflare R2 bucket and are served from our own image host (ADR-0016). Two sides read separate settings:

- API (Railway): `IMAGE_BASE_URL`, the public address of the image host with no trailing path (for example `https://img.example.com`). The API serves `IMAGE_BASE_URL` plus a row's hosted key as its image address. With it unset, or on a row with no key, the address is empty. It never falls back to the scraped source address.
- Ingester (`go run ./cmd/ingest-pokemon-asia`, run by hand, never on Railway): `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` and `R2_BUCKET` from an R2 API token scoped to that bucket. `R2_ENDPOINT` is optional and replaces the address derived from the account id. Unless all four of the first group are set, the ingester logs a warning and hosts nothing, so a local ingest of one small set needs no credentials.

Keys are `cards/<card id>` and `expansion-sets/<expansion set id>`, so re-running the ingester is the backfill: it copies only rows whose key is empty, and a failed download or upload is logged and retried by the next run. To host Expansion Set covers without crawling any card, run `go run ./cmd/ingest-pokemon-asia -sync-expansion-sets`. Hosted originals are uploaded with `Cache-Control: public, max-age=31536000, immutable`, which is safe because a key always maps to the same bytes.

Rollout order matters, because nothing enforces it: until `IMAGE_BASE_URL` is set on the API and the ingester has hosted a row, that row's image address is empty and the frontend shows its placeholder. `preserve()` in `railway.ts` keeps an existing `IMAGE_BASE_URL` but never creates one, so set it in Railway first, run the full ingester backfill, then deploy the API. The hosted-images migration also drops `image_url` in the same step as the API change, so replicas still running the old API fail on catalog queries during the rollout window.

## A failed Vercel build keeps serving the old bundle

If production shows stale behavior (missing images, old UI), suspect a failed Vercel build before suspecting the code: the previous deployment stays live. Check in this order:

1. `gh run list --limit 8`: did the CI for your merge pass?
2. `gh api repos/{owner}/{repo}/deployments --jq '.[0:4][] | [.id,.environment,.sha[0:7],.created_at] | @tsv'`: was your sha deployed?
3. `npx vercel inspect <deployment-id-or-url> --logs`: why the build failed (needs `VERCEL_TOKEN`).
4. `curl -s <api-origin>/catalog/series`: does the API itself return the data? If yes, the bug is in the frontend deploy.

## Preview environments

`.github/workflows/preview-environments.yml` provisions a stack per pull request, and its header comment is the authority on what it creates and deletes:

- A Neon branch `preview/pr-<number>`, forked from `production`.
- Neon credentials pushed into the Railway PR environment, which Railway creates and deletes itself.
- A Vercel preview deployment with a branch-scoped `VITE_API_BASE_URL` pointing at that Railway environment. Closing the PR deletes the Neon branch, the preview deployments and that env var.

Provisioning skips fork and Dependabot PRs because they get no repo secrets.
