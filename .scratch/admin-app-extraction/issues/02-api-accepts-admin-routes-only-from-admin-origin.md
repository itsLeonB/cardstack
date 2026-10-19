# 02: API accepts the admin routes only from the Admin app's origin

**What to build:** The API separates the Admin app from the user app by the origin a token was minted for (ADR-0024). A new `ADMIN_CLIENT_URLS` lists the Admin app's origins. The admin routes and the access check answer 404, with the body a non-Admin gets, unless the caller is a listed email and the token's origin is in that list. The privilege that shows an Admin unpublished Expansion Sets on the ordinary catalog routes needs the same two conditions, so a listed email on a user-origin token is an ordinary user. Ordinary routes accept a token from either list, and CORS allows the origins of both. There is no legacy mode: with `ADMIN_EMAILS` set and the list empty, the API refuses to boot.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

Do not deploy this alone: the user app's `/admin` stops working when it ships. The release order is in ticket 07.

- [ ] An Admin on an admin-origin token reaches the admin routes and the access check (204)
- [ ] The same listed email on a user-origin token gets the Guest and non-Admin 404 on every admin route, with the same body
- [ ] An unlisted email on an admin-origin token gets the same 404
- [ ] The ordinary catalog routes show unpublished Expansion Sets only to a listed email on an admin-origin token; a listed email on a user-origin token sees what a user sees
- [ ] An admin-origin token is accepted on ordinary routes, a user-origin token keeps working there, and a token from an origin in neither list is still rejected
- [ ] CORS allows the origins of both lists
- [ ] The API refuses to boot, naming the cause, when `ADMIN_EMAILS` is set and `ADMIN_CLIENT_URLS` is empty, when an entry is not a bare origin (naming the entry), and when an origin is in both `ADMIN_CLIENT_URLS` and `APP_CLIENT_URLS`
- [ ] With no allowlist the setting is not required, so previews and CI boot as before, and the admin routes are still not registered without the allowlist and the R2 settings
- [ ] The local settings example lists `ADMIN_CLIENT_URLS` with the Admin app's local origin, and the rate limits and edge secret still apply to admin traffic
- [ ] The existing admin and published-catalog route tests are updated to use admin-origin tokens, the verifier, middleware and configuration tests cover the pieces a request cannot reach, and the backend verify script passes
