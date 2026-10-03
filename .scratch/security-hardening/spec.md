# Security hardening and resilience

Status: ready-for-agent

Absorbs `.scratch/cardstack-mvp/issues/24-guest-catalog-preview.md`. Supersedes ADR-0003 and ADR-0004 (see ADR-0015) and removes the CSRF guard added by ticket 16 of the MVP effort. Images move to R2 per ADR-0016.

## Problem Statement

Cardstack's MVP was built for one user, and it shows in three places that matter once strangers can reach it.

First, sign-in is home-grown. Registration has no email verification, there is no password reset and no social login. Nothing throttles or locks out repeated password guesses on a single account. Registration tells a stranger whether an email is already registered, and login takes a different amount of time for an unknown email than for a known one.

Second, the API is wide open to anonymous callers. Anyone can page through the entire catalog, run expensive name searches and facet queries with no ceiling, and hammer the API from one address, because the one rate limit is a single global per-IP bucket that trusts a client-supplied header and so can be spoofed. The API also advertises its own documentation, accepts unbounded search input, and answers some malformed requests with echoed input.

Third, every card image and set cover is hot-linked from a third-party site. Each tile downloads a full-size PNG into a thumbnail-sized box. Pages are slower than they need to be, the app depends on someone else's server staying up and not blocking us, and we have no control over what is served.

## Solution

Make sign-in, public access and images trustworthy and cheap to operate.

- A managed identity provider (Clerk) owns sign-in: Google sign-in, email and password with mandatory email verification, forgot and reset password, lockout after repeated failures, bot protection and enumeration protection. The backend stops handling credentials, sessions and cookies entirely. It only verifies a short-lived signed token on each request and maps the caller to a user row it owns.
- A guest (a caller with no token) gets a deliberate preview of the catalog: browse Series and Expansion Sets, name search and the first page of results. Filtering by rarity, category and tag, facet counts and everything beyond the first page require an account. The API enforces this, and the UI mirrors it with visible, disabled controls and a sign-in prompt.
- Abusive traffic is limited in two layers: per-user limits inside the API for signed-in callers, and per-IP limits at the Cloudflare edge for everyone. The API only accepts traffic that came through the edge when the edge is configured. Search input is bounded, error bodies stop echoing input, and the API stops advertising its own docs in production.
- Card and Expansion Set images are copied once into our own R2 bucket and served from our own domain, resized and converted to modern formats on demand. A failed copy never falls back to hot-linking.

## User Stories

1. As a new visitor, I want to sign up with my Google account, so that I can start without choosing another password.
2. As a new visitor, I want to sign up with an email and password, so that I can use the app without a Google account.
3. As a new visitor signing up with email and password, I want to confirm my email with a code before I get in, so that nobody can create an account in my name.
4. As a visitor who signed up with an email I do not control, I want to be unable to get a session, so that the account cannot be used by the wrong person.
5. As a returning user, I want to sign in with Google or with my email and password, so that I can reach my Collections.
6. As a user who forgot my password, I want to reset it by email, so that I do not lose access to my Collections.
7. As a user, I want repeated failed sign-in attempts on my account to lock it temporarily, so that guessing my password does not work.
8. As a user, I want sign-in pages that look like the rest of Cardstack, in light and dark themes, so that the app feels like one product.
9. As a user, I want sign-in and sign-up to stay at their familiar addresses, so that bookmarks and links keep working.
10. As a user, I want to be sent back to the page I was trying to reach after signing in, so that I do not lose my place.
11. As a user, I want that return address to be restricted to pages inside Cardstack, so that a crafted link cannot send me elsewhere after signing in.
12. As a signed-in user, I want the header to show I am signed in and let me sign out, so that I can end my session on a shared device.
13. As a signed-in user, I want my session to survive a page reload and a long idle period without silently breaking requests, so that I am not interrupted mid-task.
14. As a signed-in user whose session was ended elsewhere, I want the app to notice and send me to sign in, so that I am never left on a page that fails silently.
15. As a signed-in user, I want my Collections and Inventory Entries to be reachable only by me, so that nobody else can read or change them.
16. As a signed-in user, I want requests for someone else's Collection to look exactly like requests for one that does not exist, so that nobody can discover what exists.
17. As a first-time signed-in user, I want my profile to be created automatically the first time I use the app, so that there is no separate setup step.
18. As a signed-in user, I want my display name to come from my account name, or from my email when I have no name, so that the dashboard greeting is never blank.
19. As a signed-in user, I want a changed email on my account to be reflected in Cardstack, so that what the app shows stays current.
20. As a guest, I want to browse Series and Expansion Sets, so that I can see what the catalog contains before signing up.
21. As a guest, I want to search the catalog by name and see the first page of results, so that I can check whether the cards I care about are there.
22. As a guest, I want to open an Expansion Set and see its first page of cards, so that I get a real taste of the app.
23. As a guest, I want to see filter controls for rarity, category and tag that are visibly disabled with a "Sign in to use filters" prompt, so that I know the features exist and how to unlock them.
24. As a guest who reaches the end of the first page, I want a clear "Sign in to see more" prompt where the next page would load, so that I understand why the list stops.
25. As a guest, I want the prompt to take me to sign in and bring me back to the same catalog view, so that signing in does not cost me my search.
26. As a guest, I want Add to collection and quantity controls to prompt me to sign in, so that I am never shown actions that cannot work.
27. As a signed-in user, I want the full catalog with every filter and unlimited paging, so that signing in is clearly worth it.
28. As a catalog operator, I want locked catalog features enforced by the API, not only hidden in the UI, so that a guest cannot call the endpoint directly to bypass the lock.
29. As a catalog operator, I want an over-limit or locked guest request to fail with a stable machine-readable signal, so that the UI can show the right prompt without parsing messages.
30. As a catalog operator, I want a guest's page size capped at one standard page, so that a guest cannot ask for a hundred cards at a time.
31. As a catalog operator, I want facet counts to require an account, so that anonymous callers cannot run the most expensive catalog query.
32. As a catalog operator, I want name search to accept only reasonable lengths, so that nobody can send a huge pattern that scans the table.
33. As a catalog operator, I want the number of values in each repeated filter capped, so that nobody can send thousands of filter values in one request.
34. As a catalog operator, I want signed-in users limited per user, so that one account cannot overload the API even from many addresses.
35. As a catalog operator, I want the expensive routes (name search and facets) to have tighter limits than ordinary reads, so that the costliest requests are the hardest to abuse.
36. As a catalog operator, I want a rate-limited caller to get a clear 429 response with a retry hint, so that well-behaved clients can back off.
37. As a catalog operator, I want per-IP limits enforced at the Cloudflare edge before traffic reaches the API, so that floods are absorbed cheaply.
38. As a catalog operator, I want the API to read the client address from the header Cloudflare sets and to ignore client-supplied address headers, so that nobody can dodge limits by spoofing an address.
39. As a catalog operator, I want the API to reject requests that did not come through Cloudflare when the edge secret is configured, so that the raw hosting address cannot be used to bypass the edge.
40. As a catalog operator, I want the health check to stay reachable without the edge secret, so that the host can still monitor the service.
41. As a developer, I want the edge-secret check to be off when the secret is unset, so that local development and pull-request preview environments keep working.
42. As a catalog operator, I want the existing loose per-IP limit kept inside the API as a backstop, so that a gap in the edge rules does not leave the API unprotected.
43. As a catalog operator, I want the API's interactive docs and OpenAPI document switched off in production, so that the API does not advertise its own surface.
44. As a developer, I want the committed OpenAPI file and the frontend code generation to keep working with docs off in production, so that the build is unaffected.
45. As a catalog operator, I want HSTS and a Referrer-Policy header on every API response, so that browsers keep to HTTPS and do not leak addresses.
46. As a catalog operator, I want malformed-request errors to stop echoing what the caller sent, so that error bodies cannot be used to probe or reflect input.
47. As a user, I want errors from the API to stay human-readable and free of internals, so that nothing sensitive is ever shown.
48. As a catalog operator, I want a token from another application, an expired token or a token with the wrong audience to be rejected, so that only tokens minted for Cardstack are accepted.
49. As a catalog operator, I want a token that is present but invalid to be refused with 401 rather than treated as a guest, so that the client refreshes its token instead of silently losing access.
50. As a catalog operator, I want an unsigned request to be treated as a guest only on routes that allow guests, so that private routes remain private.
51. As a user, I want card images served from Cardstack's own domain, so that pages do not depend on a third-party site.
52. As a user on a phone, I want small images for small tiles, so that the catalog loads quickly on a slow connection.
53. As a user on a high-density screen, I want sharper images when my screen needs them, so that cards look good.
54. As a user, I want images delivered in a modern format my browser supports, so that they are as small as possible.
55. As a user, I want a name placeholder when a card has no hosted image, so that a missing image never leaves a broken tile.
56. As a user on a card's detail view, I want a larger image than the tile uses, so that I can read the card.
57. As a user, I want Expansion Set covers sized for their small box, so that the set list does not download full-size art.
58. As a catalog operator, I want existing catalog rows to be moved to hosted images by re-running the ingester, so that there is no separate migration tool to learn.
59. As a catalog operator, I want the ingester to skip images already hosted, so that a re-run only copies what is missing and finishes quickly.
60. As a catalog operator, I want a failed download or upload to leave the image empty and log it, so that one bad image does not stop the run and the next run heals it.
61. As a catalog operator, I want the ingester to refuse to fetch images from any host other than the known source, so that scraped markup cannot make our servers request arbitrary addresses.
62. As a catalog operator, I want downloaded images checked to be real images of a sane size before they are stored, so that a bad response never becomes a hosted file.
63. As a catalog operator, I want the original source address kept alongside the hosted key, so that an image can be re-fetched if an upload is lost.
64. As a catalog operator, I want hosted originals served with long cache lifetimes, so that repeat views cost nothing.
65. As a catalog operator, I want a transformation quota that runs out gracefully by serving the original image, so that pages never break when the free allowance is used up.
66. As a developer, I want image hosting optional in local development, so that I can ingest a small set without R2 credentials.
67. As a developer, I want the sign-in provider swappable behind a small seam, so that tests and a future provider change do not touch handlers.
68. As a developer, I want backend tests to run with no network access to the identity provider, so that CI stays fast and stable.
69. As a developer, I want end-to-end sign-in tested through the real provider with a dedicated test user, so that the real flow is exercised.
70. As a maintainer, I want end-to-end sign-in skipped on fork pull requests, so that forks without secrets do not fail.
71. As a maintainer, I want the dead authentication code, tables, cookies and CSRF guard removed, so that there is one identity system and a smaller attack surface.
72. As the project owner, I want a step-by-step setup guide for the dashboard work only I can do (Clerk, Cloudflare, R2, secrets), so that I can finish the human parts without guessing.

## Implementation Decisions

**Identity and the backend seam**

- Clerk is the only identity system. The existing go-authkit based register, login, refresh, logout and current-user endpoints, the auth cookies and fingerprint, the CSRF guard and the go-authkit dependency are removed. ADR-0003 and ADR-0004 are marked superseded by a new ADR-0015.
- Token verification sits behind a small interface owned by the domain's HTTP layer. The production adapter verifies Clerk session tokens against Clerk's published keys, which are cached. Tests inject a fake. This follows ADR-0011: adapters for interchangeable infrastructure.
- The API authenticates by an `Authorization: Bearer` header only. There are no auth cookies, so no CSRF protection is needed and the cross-site cookie configuration goes away.
- The verifier checks signature, expiry, issuer and the authorized party (the token must have been minted for one of the configured frontend origins). The exact mechanism is confirmed at implementation against Clerk's current docs.
- Request classification has three outcomes. No `Authorization` header: guest. A valid token: authenticated, with a user and profile in context. A header that is present but invalid or expired: 401, never a silent downgrade to guest. Routes declare whether they allow guests; private routes reject guests with 401.
- Clerk's session token is configured with two custom claims, the primary email and the full name. The backend trusts these claims because the token is signed. The token stays within Clerk's 1.2KB limit.
- Email verification is enforced by Clerk: no session exists until the email is verified, so the backend adds no separate verified-email check. Google is the only social provider.
- Clerk's lockout, bot protection and bulk enumeration protection are enabled in the dashboard. Auth throttling is therefore Clerk's, not ours; this API has no auth endpoints to rate-limit.
- A development Clerk instance serves local development, pull-request previews and CI. The production instance serves the live site on the owner's domain.

**Schema and user provisioning**

- The `users` table keeps its id, gains an auth provider and an auth subject (unique together as a pair), and keeps email as a required, non-unique, indexed column. The password hash and verified flag are dropped, and so are the sessions and refresh-token tables. `user_profiles`, `collections` and everything below them are untouched.
- The migration deletes all existing `users` rows before reshaping, which cascades to profiles, collections and Inventory Entries. This is intentional ("start fresh", no live users) and runs on the production database.
- A user and profile are created lazily, in one transaction, on the first authenticated request, safe under concurrent first requests. Profile name comes from the name claim, falling back to the local part of the email, because the name is empty for plain email signups. Name is written once and never overwritten, since the profile is domain-owned display info.
- On later requests, the stored email is updated only when the claim differs from it.
- A short in-memory cache maps the auth identity to the profile so most requests do not touch the database. The existing session-cache module is the starting point.
- The current-user endpoint is removed. The frontend learns who is signed in from Clerk and the first authenticated call creates the profile.
- Deleting an account in Clerk leaves our rows behind. Syncing deletions needs a Clerk webhook and is out of scope.

**Guest preview (absorbs ticket 24)**

- Public catalog list routes accept guests. A guest sees Series, Expansion Sets, rarities, categories and tags lists, name search and filtering by Expansion Set. A guest receives only the first page of results, and a requested page size above one standard page (24) is reduced to 24.
- For a guest, any page after the first, any filter on rarity, category or tag, and the facets endpoint answer 401 with a stable machine-readable `login_required` code, classified through the existing error taxonomy of ADR-0013.
- Result metadata for a guest still reports the true total, so the UI can say how much there is behind the lock.
- Signed-in callers get the full behavior, including the existing page size ceiling of 100.
- The lock is enforced in the API. The UI renders locked filter controls disabled with a sign-in prompt, shows "Sign in to see more" where an infinite grid would load the next page, and shows a sign-in prompt in place of Add to collection and quantity controls. Sign-in from any prompt returns the user to the same catalog view.

**Request hardening**

- Name search accepts at most 64 characters. Each repeated filter (Expansion Set, rarity, category, tag) accepts at most 20 values. Violations are ordinary validation errors.
- Error responses for malformed requests no longer echo caller-supplied values, including the invalid-identifier message in the card filter builder and the detail strings Huma produces for decode errors. This extends ADR-0013's redaction rule to client errors.
- Every API response carries HSTS and a Referrer-Policy header, in addition to the headers already set.
- The interactive docs page and the OpenAPI endpoints are disabled when the app runs in production. The committed OpenAPI file remains the input to frontend code generation.
- CORS behavior is unchanged: allowed origins come from configuration, with the existing fallback behavior.
- There is no cache on the facets endpoint in this effort.

**Rate limiting and the edge**

- Signed-in callers are limited per user (keyed by the verified user identity) inside the API. Name search and facets have their own tighter buckets. Exceeding a limit returns 429 with a retry hint, classified through the error taxonomy. Exact numbers are configuration with defaults chosen at implementation and recorded in the ticket.
- Per-IP limiting is a Cloudflare rule on the API's own domain, proxied through Cloudflare. The free plan allows one rule on a short window, so the rule is a coarse flood limiter and the in-API per-IP limit stays as a loose backstop.
- The API takes the client address from the header Cloudflare sets and ignores `X-Real-IP` and other client-supplied address headers whenever the edge secret is configured. This closes the current spoofable-address gap.
- An optional edge secret closes the bypass through the raw hosting address. Cloudflare adds a secret request header; when the secret is configured the API rejects any request without it, except the health check. When the secret is unset the check is off, so local development and preview environments (which have their own hosting URLs and no Cloudflare) keep working.
- Per-email and per-account throttling is not built. Cloudflare cannot key on it below Enterprise, and Clerk owns it.

**Frontend identity**

- The Clerk provider wraps the app, which stays an SPA (ADR-0014 is not reopened). Sign-in and sign-up pages remain at their current addresses and render Clerk's prebuilt components themed to the visual identity.
- The generated API client's shared request wrapper attaches the current Clerk token as a bearer header on every call. On a 401 it asks Clerk for a fresh token and retries once; a second 401 triggers the existing auth-lost handling. The cookie, CSRF-token and refresh-on-401 code, and the current-user query, are removed.
- Signed-in state for the shell, route guards and redirects derives from Clerk. The redirect-after-login behavior and its same-origin path check, and the redirect of signed-in users away from guest-only pages, are preserved.
- The Vercel build gains Clerk's publishable key. A user's name and email shown in the shell come from Clerk's user data.

**Images on R2 (ADR-0016)**

- The ingester downloads each card image and each Expansion Set cover from the known source host and uploads the original, unmodified, to an R2 bucket. Keys are deterministic from the row's identifier, one namespace for cards and one for Expansion Sets. Re-running the ingester skips keys that already exist, so a re-run is the backfill.
- Cards and Expansion Sets gain a hosted-image key column. The scraped source address is kept in its own column. The API's image address field is now built from a configured image base address plus the key, and is empty when there is no key.
- Failure handling: a failed download or upload leaves the key empty and logs the failure. The run continues, a later run retries, and the API never falls back to the source address.
- SSRF and integrity guards: the ingester fetches images only from the known source host, enforces a maximum size, and checks the content really is an image before upload.
- Hosted originals are served from a Cloudflare-proxied subdomain with long-lived cache headers. Transformations run at read time through that subdomain: tile widths 240 and 480 (for 1x and 2x screens), detail view 720, Expansion Set cover 128, with automatic format selection and a redirect to the original when the transformation allowance is exhausted.
- A frontend image helper turns the API's image address into the right sized, formatted address and a responsive source set. When the address is not on the configured image host (local development, or no hosting configured), it returns the address unchanged. The existing name-placeholder fallback on image load errors stays.
- When R2 is not configured, the ingester logs a warning and skips hosting, so local ingestion of a small set needs no credentials.

**Configuration**

- New backend settings: Clerk instance configuration, the allowed frontend origins for the authorized-party check, the optional edge secret, R2 credentials and bucket, the image base address, and the rate-limit tiers.
- New frontend settings: Clerk's publishable key and the image host.
- Preview environments use the Clerk development instance and no edge secret.

## Testing Decisions

A good test exercises external behavior through the highest existing seam: an HTTP request in, a status and body out; an ingest run in, database rows and stored objects out; a rendered route in, what a person sees out. Tests do not assert on private helpers, internal call order or which library did the work.

There are four seams. They reuse the project's existing test layers (general conventions: unit tests own branches, feature and e2e tests own flows).

1. **Backend HTTP route tests** (the existing route test layer) with a fake token verifier and a real Postgres, per ADR-0005. This one seam covers: guest versus authenticated versus invalid-token behavior on each route, the guest lock (page, filters, facets, page-size clamp, the `login_required` code), ownership isolation (foreign and missing Collections look identical), lazy user and profile creation including concurrent first requests, email refresh and name immutability, input limits, security headers, docs-off in production, error redaction, per-user rate limiting including the tighter buckets, and the edge-secret and client-address behavior. Branch-level detail (limiter arithmetic, claim parsing) gets unit tests next to the unit. The user repository tests are rewritten for the new table shape and the session and refresh-token repository tests are deleted with their code.
2. **Ingester tests** (the existing ingest test layer) with a fake object-store interface and a local test server standing in for the image source, over a real Postgres. They cover upload and key assignment, skip-if-hosted, failure leaving an empty key without aborting the run, the host allow-list, size and content checks, and the no-R2-configured path.
3. **Frontend feature tests** with the generated API client and Clerk's hooks mocked, matching the existing approach of mocking the generated client module. They cover guest versus signed-in rendering of the catalog (disabled filters with prompt, end-of-list prompt, Add to collection prompt), the bearer-token and retry-once behavior of the request wrapper, route guards and redirects, and the image helper (sized and unsized cases, empty address, off-host address, load-error placeholder).
4. **A thin Playwright end-to-end layer** for sign-in only, using Clerk Testing Tokens with a dedicated test user in the development instance. It covers the sign-in happy path and one key failure, not every branch again. It needs the Clerk secret key and the test user's credentials as repository secrets and is skipped on fork pull requests, as the preview-environment workflow already skips them. Existing e2e specs that registered a user through the old form are updated to sign in this way.

Prior art: the existing backend route and ingest tests, the existing frontend feature tests that mock the generated client, and the existing end-to-end specs.

## Out of Scope

- Syncing account deletion from Clerk to our database (needs a webhook).
- Other social providers (Discord, Apple, GitHub) and multi-factor authentication beyond what Clerk enables by default.
- A cache on the facets endpoint.
- Changing CORS behavior.
- Per-email or per-account throttling in our own code.
- Series images (still parked in ticket 15).
- Pre-generated image variants, an image proxy of our own, or hosting images for any source other than the current one.
- Reopening ADR-0014 (client-side rendering and SEO). Clerk does not force a change.
- Cloudflare Enterprise features, and a Redis-backed or shared rate limiter. The in-memory limiter is correct for a single replica.
- Guest-visible changes beyond the catalog preview (collections, inventory and the dashboard remain signed-in features).

## Further Notes

**Human setup (ticket for the owner, ideally as an interactive wizard).** The following cannot be done by an agent and gates other work.

- Clerk dashboard: create the development and production instances, enable Google, require email verification for email and password, enable forgot and reset password, enable lockout and enumeration protection and bot protection, and add the email and name claims to the session token. Some of these settings may be plan-gated; this was not confirmed.
- Cloudflare: put the owner's domain on Cloudflare, create the R2 bucket and its public subdomain, enable image transformations on that zone, proxy the API subdomain, add a Transform Rule that sets the secret header, and add the coarse per-IP rate rule.
- Secrets and variables: Clerk keys, the edge secret, R2 credentials and the image base address on Railway; Clerk's publishable key and the image host on Vercel; the Clerk secret key and test-user credentials as GitHub secrets for end-to-end.
- The production Clerk instance needs a domain the owner controls; a `*.vercel.app` address will not work for production.

**Assumptions not verified in the grilling session.** Railway custom domains work behind Cloudflare's proxy in "Full" SSL mode; Clerk's SDK for TanStack Start works in the SPA mode this app uses (confirm at implementation); exact Clerk plan limits for the security features above.

**Free-tier ceilings worth knowing.** Cloudflare's free plan allows 5,000 unique image transformations per month; beyond that new variants return an error and are not billed, and the redirect-to-original fallback keeps pages working. Only images actually viewed count, and the guest page cap limits how fast a scraper can burn the allowance. The free per-IP rule allows a single rule on a 10-second window.

**Suggested order.** Image hosting, request hardening and the Clerk backend have no dependencies and can proceed in parallel. The Clerk frontend follows the Clerk backend. The guest lock API and per-user rate limits need the Clerk backend; the guest lock UI needs the lock API and the Clerk frontend. End-to-end sign-in needs the Clerk frontend. Production cutover should happen after the human setup is done and the migration's data wipe is accepted once more at deploy time.

**Contradictions to be aware of.** This effort deletes the CSRF guard ticket 16 added and supersedes ADR-0003 and ADR-0004. ADR-0014's stated reason for client-side SEO (HttpOnly session cookies unreadable by the server) no longer holds, but its conclusion is not reopened here.
