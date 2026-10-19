# Admin app

The Admin app is the Admin's own web app, on its own host and built like `frontend/` (ADR-0024). It shows the ordinary not-found page to everyone who is not an Admin, so it has no sign-in page of its own: an Admin signs in through the user app and the session carries over.

## Run locally

1. Start the API with `ADMIN_CLIENT_URLS=http://localhost:3001` (see `backend/.env.example`).
2. Sign in at `http://localhost:3000` with a listed email.
3. Copy `.env.example` to `.env` and run `bun run dev`, then open `http://localhost:3001`.

## Checks

Run `scripts/verification/frontend.sh` from the repo root, which covers this package and `frontend/`.

`bun run codegen` regenerates `src/generated/` from `backend/openapi.json`. It generates only the operations listed in `orval.config.ts`, so add an operationId there before the Admin app calls it.

`bun run build` fails in a production build (`VERCEL_ENV=production`) when `VITE_CLERK_PUBLISHABLE_KEY` or `VITE_API_BASE_URL` is unset, and only warns elsewhere.
