# Backend sign-in (Clerk)

The API authenticates only by an `Authorization: Bearer` Clerk session token (ADR-0015); it holds no credentials, sessions or cookies. It verifies the signature against the instance's keys (fetched with the secret key and cached), the expiry, the issuer, and that the token's authorized party is one of the frontend origins. The API refuses to boot without the secret key and issuer (the migration job and the ingesters do not need them), and rejects every token (logging why) while no frontend origin is configured. Local values: `backend/.env.example`.

- `CLERK_SECRET_KEY`: the instance's secret key, used to fetch its signing keys. Keep it out of the frontend.
- `CLERK_ISSUER`: the instance's Frontend API URL, the `iss` claim of its tokens (for example `https://example.clerk.accounts.dev`). It is also the base64 payload of the publishable key, which is how the preview and end-to-end workflows derive it.
- `APP_CLIENT_URLS`: the frontend origins, comma-separated, also used for CORS. A token minted for any other origin is rejected with 401, and the match is an exact string comparison: list every origin the site is served from (`https://www.example.com` and `https://example.com` are two). The API refuses to boot with an entry that is not a bare origin (a path, trailing slash, space after a comma, quotes, or an empty entry from a trailing comma) and names the entry in the error; an empty list still boots. It logs `refusing token: ... authorized party "<origin>"` for a token whose origin is not listed, so a 401 with a valid-looking token is diagnosable from the Railway logs. The preview workflow sets it to the Vercel preview URL after deploying it.

Rollout order matters, because nothing enforces it. `preserve()` in `railway.ts` keeps an existing `CLERK_SECRET_KEY` and `CLERK_ISSUER` but never creates them, and the migration runs before the new API starts. Set `CLERK_SECRET_KEY`, `CLERK_ISSUER` and `APP_CLIENT_URLS` (the production frontend origin) in Railway production before merging, or the migration deletes every user and drops the old auth tables while the new API exits at boot and the old one keeps running against the changed schema.

The Clerk session token must carry the `email` and `name` custom claims (`scripts/clerk-setup.sh` adds them). A token without an email claim is rejected.

The first authenticated request creates the user and profile; the migration that introduced this deletes every existing user, with their Collections and Inventory Entries, once, when it first runs on a database.
