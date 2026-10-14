# Backend sign-in (Clerk)

The API authenticates only by an `Authorization: Bearer` Clerk session token (ADR-0015); it holds no credentials, sessions or cookies. It verifies the signature against the instance's keys (fetched with the secret key and cached), the expiry, the issuer, and that the token's authorized party is one of the frontend origins. The API refuses to boot without the secret key and issuer (the migration job and the ingesters do not need them), and rejects every token (logging why) while no frontend origin is configured. Local values: `backend/.env.example`.

- `CLERK_SECRET_KEY`: the instance's secret key, used to fetch its signing keys. Keep it out of the frontend.
- `CLERK_ISSUER`: the instance's Frontend API URL, the `iss` claim of its tokens (for example `https://example.clerk.accounts.dev`). It is also the base64 payload of the publishable key, which is how the preview and end-to-end workflows derive it.
- `APP_CLIENT_URLS`: the frontend origins, comma-separated, also used for CORS. A token minted for any other origin is rejected with 401, and the match is an exact string comparison: list every origin the site is served from (`https://www.example.com` and `https://example.com` are two). The API refuses to boot with an entry that is not a bare origin (a path, trailing slash, space after a comma, quotes, or an empty entry from a trailing comma) and names the entry in the error; an empty list still boots. It logs `refusing token: ... authorized party "<origin>"` for a token whose origin is not listed, so a 401 with a valid-looking token is diagnosable from the Railway logs. The preview workflow sets it to the Vercel preview URL after deploying it.

## Admins (`ADMIN_EMAILS`)

`ADMIN_EMAILS`: the emails allowed to edit the catalog (ADR-0019), comma-separated, for example `owner@example.com,second@example.com`. There is no admin role or account type: a caller is an Admin when the token's `email` claim equals an entry, compared case-insensitively. The API refuses to boot with an entry that is not a bare email address (a name, spaces, quotes, or an empty entry from a trailing comma) and names the entry in the error. Unset, the admin routes do not exist.

- The admin routes (`/admin/...`) are registered only when `ADMIN_EMAILS` and all four `R2_*` settings are set on the API (`docs/agents/deployment/images.md`). If only one side is set, the API boots without them and logs that the allowlist is set but the R2 settings are not. Previews and CI have neither, so they have no admin surface.
- Every admin route answers 404 to a Guest and to a signed-in caller who is not listed, with the same body, so the surface does not reveal itself. Admin traffic goes through the same edge secret and per-user rate limits as the rest of the API.
- The allowlist is only as strong as Clerk's email verification. Keep the instance requiring a verified email at sign-up (it has from the start, ADR-0015): turning that off would let anyone claim a listed address.
- The frontend `/admin` page is code-split, linked from nowhere and absent from the sitemap; to a non-Admin it shows the ordinary not-found page.

Rollout order matters, because nothing enforces it. `preserve()` in `railway.ts` keeps an existing `CLERK_SECRET_KEY` and `CLERK_ISSUER` but never creates them, and the migration runs before the new API starts. Set `CLERK_SECRET_KEY`, `CLERK_ISSUER` and `APP_CLIENT_URLS` (the production frontend origin) in Railway production before merging, or the migration deletes every user and drops the old auth tables while the new API exits at boot and the old one keeps running against the changed schema.

The Clerk session token must carry the `email` and `name` custom claims (`scripts/clerk-setup.sh` adds them). A token without an email claim is rejected.

The first authenticated request creates the user and profile; the migration that introduced this deletes every existing user, with their Collections and Inventory Entries, once, when it first runs on a database.
