# Admin app extraction

Status: ready-for-agent

## Problem Statement

The Admin writes catalog content through an `/admin` page inside the user app (ADR-0019). It is a small lazy-loaded page today, but it is the wrong place for an Admin's work to grow: the Admin's code ships in the user app's source and route tree, an Admin's session token from the user app can call every admin route, and an Admin who browses the user app sees unpublished Expansion Sets there. The photo review planned in the user-supplied embeddings spec, and the photos that ADR-0022 stores, will make the Admin's job larger. The operator wants the Admin to be its own app, kept apart from the user app, without a second backend or a second identity system.

## Solution

The Admin gets its own web app, the Admin app (GLOSSARY.md): a separate package built like the user app, deployed as its own project on a sibling subdomain of the production domain, not linked from the user app and not indexed. It uses the same API, the same database and the same Clerk sign-in, and the same email allowlist decides who is an Admin. The API now accepts the admin routes only from tokens minted for the Admin app's origin, so an Admin's token from the user app is refused with the same 404 as a non-Admin, and the privilege of seeing unpublished Expansion Sets on ordinary catalog routes needs the same admin-origin token. The user app loses `/admin`. The first version of the Admin app has the same screens as today and no new ones. The decision and its alternatives are in ADR-0024.

## User Stories

1. As an Admin, I want to open the Admin app on its own address, so that my catalog work is not a corner of the app users sign in to
2. As an Admin, I want to sign in to the Admin app with the same Clerk account I use elsewhere, so that I do not need a second login
3. As an Admin, I want to be already signed in to the Admin app when I am signed in to the user app on the same domain, so that I do not sign in twice
4. As an Admin, I want the Admin app to show the ordinary not-found page to a Guest, so that its existence is not revealed
5. As an Admin, I want the Admin app to show the ordinary not-found page to a signed-in user who is not listed, so that its existence is not revealed
6. As an Admin, I want a failed access check (a rate limit or a server error) to show an error and not the not-found page, so that a real failure is not mistaken for no access
7. As an Admin, I want to create an Expansion Set in the Admin app exactly as I did in the user app, so that nothing about my work changes
8. As an Admin, I want to edit an Expansion Set in the Admin app with every field sent back, so that the replace behaviour of ADR-0019 stays
9. As an Admin, I want to add a Card with its image in the Admin app, so that I can enter content with no upstream source
10. As an Admin, I want the image checks (type, size, dimensions) and the preview before saving to work as before, so that I catch a bad file early
11. As an Admin, I want fast card entry (save and next) to work as before, so that entering a whole set stays quick
12. As an Admin, I want to edit and delete a Card in the Admin app with the same confirmation that names what goes, so that I cannot delete by accident
13. As an Admin, I want a Card to be found by Expansion Set and number as before, so that the edit form loads the existing Card
14. As an Admin, I want to see unpublished Expansion Sets in the Admin app, so that I can check and publish content before users see it
15. As an Admin, I want the Admin app to read the ordinary catalog routes too, so that its pickers and lists use the same data users see plus the unpublished sets
16. As an Admin, I want the image upload from the Admin app to reach R2 directly with the presigned address, so that uploads work from the new origin
17. As an Admin, I want a card I curate to stay Curated and untouched by the ingesters, so that extraction changes nothing about who owns a row
18. As an Admin, I want the Admin app to use the same look as the user app, so that I do not learn a second interface
19. As an Admin, I want an Admin app link in the user app nowhere, so that no user is pointed at it
20. As an Admin, I want a failed upload or a failed confirm to say what failed, as before, so that I know whether to retry
21. As a signed-in user who is listed as an Admin, I want the user app to treat me as an ordinary user, so that what I see there is what everyone sees
22. As a signed-in user who is listed as an Admin, I want the user app to hide unpublished Expansion Sets from me, so that I cannot record holdings against content that is not ready
23. As a user, I want the user app to have no `/admin` page, so that the address answers the ordinary not-found page like any unknown address
24. As a user, I want the user app to load a smaller bundle than before, so that it carries no admin code
25. As a user, I want the binder layout form to look and work exactly as before after the shared style constant moves out of the admin code, so that the extraction does not change my screen
26. As a user, I want sign-in, collections, inventory and scanning to work exactly as before, so that the extraction is invisible to me
27. As the operator, I want the Admin app on a sibling subdomain of the production domain, so that the Clerk session is shared with the user app
28. As the operator, I want the Admin app answered with `noindex`, a `robots.txt` that disallows everything and no sitemap, so that search engines do not list it
29. As the operator, I want the Admin app deployed as its own Vercel project, so that it builds and deploys on its own
30. As the operator, I want the Admin app to build with the production Clerk key and the API address baked in, and the build to fail in production if either is missing, so that I never ship an app that cannot sign in or calls localhost
31. As the operator, I want a new `ADMIN_CLIENT_URLS` setting that lists the Admin app's origins, so that the API knows which origin is the Admin app
32. As the operator, I want the API to refuse to boot when `ADMIN_EMAILS` is set and `ADMIN_CLIENT_URLS` is empty, so that there is no insecure default
33. As the operator, I want the API to refuse to boot when an `ADMIN_CLIENT_URLS` entry is not a bare origin, naming the entry, so that a typo does not silently reject every admin token
34. As the operator, I want the API to refuse to boot when one origin is in both `APP_CLIENT_URLS` and `ADMIN_CLIENT_URLS`, so that a token's origin is never ambiguous
35. As the operator, I want the admin routes to answer 404 to a token minted for any origin outside `ADMIN_CLIENT_URLS`, with the same body as for a non-Admin, so that a stolen user-origin token reveals nothing
36. As the operator, I want a valid Admin token from the Admin app to be accepted on the admin routes, so that I can work
37. As the operator, I want a token from the Admin app to be accepted on the ordinary routes too, so that the Admin app can read the catalog
38. As the operator, I want a token from the user app to keep working on the ordinary routes, so that users are not affected
39. As the operator, I want the Admin to see unpublished Expansion Sets on the ordinary catalog routes only with an admin-origin token, so that script in the user app cannot read unpublished content
40. As the operator, I want CORS to allow the origins of both lists, so that both browsers can call the API
41. As the operator, I want admin traffic to keep going through the same edge secret and per-user rate limits, so that protection does not weaken
42. As the operator, I want the admin routes not to exist when `ADMIN_EMAILS` or the R2 settings are missing, as before, so that previews and CI have no admin surface
43. As the operator, I want R2 CORS to allow PUT from the Admin app's origin, so that uploads from the new host succeed
44. As the operator, I want a documented deploy order (set `ADMIN_CLIENT_URLS` in Railway, deploy the API, deploy the Admin app, deploy the user app without `/admin`), so that the cutover is a checklist
45. As the operator, I want to roll back by reverting the release, so that a bad cutover has a known way back
46. As the operator, I want the local Admin app to run on its own port against the local API and the development Clerk instance, with that origin in `ADMIN_CLIENT_URLS`, so that I can develop it
47. As the operator, I want the deployment docs, the Clerk doc and the images doc updated, so that the next person finds the new setting and the new host
48. As a developer, I want a CI job for the Admin app (typecheck, lint, tests, build) that is a required check, so that it cannot break unnoticed
49. As a developer, I want the Admin app's API client generated from the same OpenAPI document, limited to the operations it uses, so that it stays in step with the backend
50. As a developer, I want the user app's generated client to leave out the admin operations, so that the user bundle holds none of them
51. As a developer, I want the Admin app to follow the same frontend conventions as the user app, so that I write code the same way in both
52. As a developer, I want the agent definitions and the orchestration doc to cover the new package, so that a frontend agent can work in it
53. As a developer, I want the moved admin tests to keep passing in the Admin app, so that I know the move lost nothing
54. As a developer, I want the user-supplied embeddings spec to build its review screens in the Admin app, so that no new admin screen lands in the user app

## Implementation Decisions

- The Admin app is a new top-level package built like the user app: TanStack Start in SPA mode, Vite, Tailwind 4, orval, bun, oxlint and vitest, with its own copy of the shadcn components and the design tokens it needs. There is no shared workspace or shared package (ADR-0024).
- Its API client is generated by its own orval target from the same OpenAPI document, limited to the admin operations and the catalog operations the Admin app calls. The user app's orval target leaves the admin operations out, so the user bundle does not hold them. The same custom fetch wrapper and base-URL fallback pattern apply.
- Moved from the user app, unchanged in behaviour: the admin gate and page, the Expansion Set form, the Card form with fast entry and image preview, the delete-card form, the shared form helpers, the upload helper and all their tests. The access check still asks the API and treats 204 as an Admin, 404 as not found, and anything else as an error.
- The user app then loses the `/admin` route, its components, tests and helpers. The one user-app file that imported a style constant from the admin form helpers gets its own copy of it.
- Auth configuration gains `ADMIN_CLIENT_URLS`, validated like `APP_CLIENT_URLS` (bare origins only, the offending entry named). The API refuses to boot when `ADMIN_EMAILS` is set and the list is empty, when an entry is not a bare origin, and when an origin is in both lists. With no allowlist the setting is not required, so previews and CI boot as before.
- The token verifier accepts a token whose origin is in either list and records, on the verified identity, whether that origin belongs to the Admin app. Tokens with no origin, or an origin in neither list, are rejected as today.
- The admin-only gate answers 404, with the body a non-Admin gets, unless the caller is a listed email and the token's origin is an Admin app origin. The privilege that lets an Admin see unpublished Expansion Sets on ordinary catalog routes requires the same two conditions. A listed email on a user-origin token is an ordinary user everywhere.
- The access check route stays and uses the same gate. CORS allows the union of both lists' origins. The rate limits, the edge secret and the registration rule (admin routes exist only when the allowlist and the R2 settings are set) are unchanged.
- No schema change, no new route and no change to any admin request or response body. The OpenAPI document changes only if a route's description changes.
- The Admin app deploys as its own Vercel project on a sibling subdomain of the production domain. It serves `noindex`, a `robots.txt` that disallows everything and no sitemap. Its build needs the Clerk publishable key (the same instance as the user app) and the API address, and fails in production when either is missing, as the user app's build does. The production hostname is chosen at deploy time and recorded in the deployment docs without the domain.
- The Clerk session is shared across subdomains of one production domain. The development instance uses a different local port for the Admin app, and that origin goes in `ADMIN_CLIENT_URLS` in the local settings example.
- The R2 image bucket's CORS gains the Admin app's origin for PUT. This is a manual operator step listed in the deploy order.
- Cutover is direct (ADR-0024): set `ADMIN_CLIENT_URLS` in Railway, deploy the API, deploy the Admin app, deploy the user app without `/admin`. The admin routes are unreachable for an Admin from when the API ships until the Admin app is live. Rollback is a revert.
- A CI job for the Admin app runs typecheck, lint, tests and build on changes to it. No Playwright spec and no per-PR preview environment is added for it, and the preview workflow stays as it is.
- Documentation changes: the deployment doc (topology, env vars, the cutover order), the Clerk doc (`ADMIN_CLIENT_URLS`, the admin-origin rule and how the Admin app is protected), the images doc (R2 CORS), the code conventions (the frontend conventions apply to both packages), the orchestration doc and the frontend agent definition (so the package is in its scope), and the repository README if it lists the packages.

## Testing Decisions

A good test exercises external behaviour only: a request with a token and the response it gets at the backend, a rendered screen and user interaction at the frontend. It does not assert how the origin is stored on the identity or which helper the gate calls.

Seams, preferring existing ones:
- **Backend: the Huma HTTP boundary against real Postgres** (ADR-0005), through the existing route test environment. It covers the new rule from the outside: an Admin on an admin-origin token reaches the admin routes and the access check; the same email on a user-origin token gets the Guest and non-Admin 404 on every admin route; an unlisted email on an admin-origin token gets the same 404; the ordinary catalog routes show unpublished Expansion Sets only to a listed email on an admin-origin token; an admin-origin token reads the ordinary routes; a user-origin token keeps working; a token from an origin in neither list is rejected. A few small tests under it cover the pieces that cannot be reached by a request: the boot refusals for configuration (missing list with an allowlist set, a non-origin entry, an origin in both lists), as the existing configuration tests do, and the verifier accepting either list.
- **Frontend, Admin app: rendered screens with the generated client mocked** (the existing Vitest and Testing Library pattern). The existing admin tests move over with the code (the Expansion Set form, the Card form, the delete form, the upload helper and the route gate) and must pass unchanged in behaviour. This is a move, not a new seam.
- **Frontend, user app: the existing suite.** A check that `/admin` now answers the not-found page, and the binder layout form tests still passing, cover the removal.
- No new Playwright spec. The existing end-to-end suite keeps passing, since it never used the admin.

Prior art: the admin route tests and the published-catalog route tests, the verifier and middleware tests in the auth package, the configuration tests, and the admin component and upload tests that move.

## Out of Scope

- Reading Corrections in the Admin app. The operator reads them through the tuning script today (scan enhancements, ticket 05).
- The review of user-supplied photos, and any new Admin screen. They belong to the user-supplied embeddings spec, which builds them in the Admin app.
- Per-PR preview environments, Playwright coverage and Clerk satellite domains for the Admin app.
- A shared workspace or shared package between the two frontends, a second Go service, a second Clerk instance and an edge login in front of the Admin app (ADR-0024).
- A staggered cutover or a legacy mode.
- Any change to how the user app scans, to the Admin's request and response bodies, or to Curated and Source rules.
- A stored Admin role.

## Further Notes

- This is the second of three specs, in the order: scan enhancements, admin app extraction, user-supplied embeddings. The third now has its home: the Admin app.
- The production domain is not recorded in the repository. Choosing the Admin app's hostname and adding it to Clerk, Railway and R2 are operator steps at deploy time.
- A production Clerk instance only shares a session across subdomains of one domain. If the user app ever moves to another domain, the Admin app needs a Clerk satellite domain.
- The two frontends duplicate the shadcn components and the design tokens. A change to one has to be made in the other; that cost is accepted in ADR-0024.
- Suggested ticket order: the API rule and configuration, the Admin app package and its moved screens, the CI job, the user-app removal, then the documentation and deployment steps.
