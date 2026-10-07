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
- `VITE_SCAN_ENABLED`: shows the "Scan cards" button and the `/collections/:id/scan` route only when it is exactly `true`. Leave it unset in Vercel Production until the real matcher has landed and been checked (the match endpoint returns random cards until then); enabling it is a frontend rebuild. Local value: `frontend/.env.example`.

## Backend settings, one file each

- `docs/agents/deployment/clerk.md`: sign-in fails or every request gets a 401, `CLERK_SECRET_KEY`, `CLERK_ISSUER`, `APP_CLIENT_URLS`, the rollout order of the Clerk migration.
- `docs/agents/deployment/images.md`: card or Expansion Set images missing or empty, `IMAGE_BASE_URL`, `VITE_IMAGE_HOST`, the R2 settings, `make host-images`.
- `docs/agents/deployment/api-protection.md`: the API answers 403 or 429, `APP_EDGE_SECRET`, the Cloudflare rule, `RATE_LIMIT_*`.

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
