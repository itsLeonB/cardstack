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

## Backend sign-in (Clerk)

The API authenticates only by an `Authorization: Bearer` Clerk session token (ADR-0015); it holds no credentials, sessions or cookies. It verifies the signature against the instance's keys (fetched with the secret key and cached), the expiry, the issuer, and that the token's authorized party is one of the frontend origins. It refuses to boot without the secret key and issuer, and rejects every token (logging why) while no frontend origin is configured. Local values: `backend/.env.example`.

- `CLERK_SECRET_KEY`: the instance's secret key, used to fetch its signing keys. Keep it out of the frontend.
- `CLERK_ISSUER`: the instance's Frontend API URL, the `iss` claim of its tokens (for example `https://example.clerk.accounts.dev`). It is also the base64 payload of the publishable key, which is how the preview and end-to-end workflows derive it.
- `APP_CLIENT_URLS`: the frontend origins, comma-separated, also used for CORS. A token minted for any other origin is rejected with 401. The preview workflow sets it to the Vercel preview URL after deploying it.

The Clerk session token must carry the `email` and `name` custom claims (`scripts/clerk-setup.sh` adds them). A token without an email claim is rejected.

The first authenticated request creates the user and profile; the migration that introduced this deletes every existing user, with their Collections and Inventory Entries, once, when it first runs on a database.

## Backend edge secret

- `APP_EDGE_SECRET` (Railway production): when set, the API rejects every request without a matching `X-Edge-Secret` header (403, except `/health`) and takes the client address from `CF-Connecting-IP`, ignoring `X-Forwarded-For` and `X-Real-IP`. Configure a Cloudflare Transform Rule on the API's proxied domain that adds `X-Edge-Secret: <value>` to every request.
- Leave it **unset in preview environments** (and locally): they are reached through their own hosting URLs, not Cloudflare, so a set secret would reject every request. When unset, no edge check runs and the client address comes from `X-Real-IP` as before.

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
