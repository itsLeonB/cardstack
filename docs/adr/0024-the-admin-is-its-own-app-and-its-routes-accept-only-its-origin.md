Status: supersedes the "admin area inside the existing app" part of [ADR-0019](0019-catalog-content-is-curated-through-an-in-app-admin.md); the allowlist, the routes, the upload flow and curated rows stand.

# The Admin is its own app, and the API accepts the admin routes only from its origin

The Admin's job is growing (photo review is planned, ADR-0022 stores the photos), and a page inside the user app puts admin code in the same bundle, behind the same origin and the same session token as every user. So the Admin gets its own app, the Admin app (GLOSSARY.md): a top-level `admin/` package built like `frontend/` (TanStack Start in SPA mode, its own orval target reading the same `openapi.json`, copied shadcn components), deployed as its own Vercel project on a sibling subdomain of the production domain, with `noindex`, a `robots.txt` Disallow and no sitemap. It is the same API, database, R2 credential and Clerk instance, and `ADMIN_EMAILS` still decides who is an Admin (ADR-0015, ADR-0019). The user app loses `/admin`.

The separation is enforced by the API, not by CORS. A new `ADMIN_CLIENT_URLS` lists the Admin app's origins. A token carries the origin it was minted for (`azp`), and the admin routes accept only a token whose `azp` is in that list: an Admin's token from the user app answers 404, with the same body as for a non-Admin. Ordinary routes accept a token from either list, because the Admin app reads the catalog too. The privilege that shows an Admin unpublished Expansion Sets on ordinary catalog routes (`AdminFlag`) needs the same admin-origin token, so in the user app an Admin is an ordinary user. CORS allows both lists' origins; it only keeps other browser origins out and is not the guarantee. The point is that script injected into the user app can mint only user-origin tokens, which are useless against `/admin/*` and against unpublished content.

The cutover is direct, with no legacy mode. When `ADMIN_EMAILS` is set, the API refuses to boot without `ADMIN_CLIENT_URLS`. One release deploys the API with the variable already set in Railway, then the Admin app, and removes `/admin`, its components, tests and upload helper from the user app. There is a short window, from the API deploy until the Admin app is live, where no Admin can write; rollback is a revert.

## Considered Options

- Keep `/admin` in the user app and let it grow. Rejected: it keeps admin code in the user bundle and the user origin's tokens able to reach admin routes.
- A second Go service for the admin routes. Rejected: it duplicates wiring (providers, config, edge secret, rate limits) and adds no capability the origin check does not give.
- A separate Clerk instance for Admins. Rejected: a second user pool next to the one ADR-0015 made the only identity system, for a threat the `azp` check already covers.
- A shared workspace package for the client and UI (or one package building two apps). Rejected: shadcn components are copied by design, the client is regenerated from one spec, and a workspace would be a refactor of `frontend/` for one operator.
- An edge login (Cloudflare Access, Vercel deployment protection) in front of the Admin app. Rejected: the shell holds no secret and the API already answers 404 to non-Admins, so it would add a second login.
- A staggered cutover with a legacy mode (the admin routes accepting user-origin tokens until a flag flips). Rejected by the operator: one operator can take a short window, and legacy mode leaves an insecure default to remove later.

## Consequences

`ADMIN_CLIENT_URLS` must be set in Railway before the API is deployed, or it will not boot, as with the Clerk settings (docs/agents/deployment/clerk.md). The admin subdomain must share the Clerk session with the user app's host, which a production instance only does for subdomains of one domain; a different domain would need a Clerk satellite domain. R2 CORS must also allow PUT from the Admin app's origin. Previews and CI have no Admin app and no allowlist, as the admin has none today. The two apps duplicate the shadcn components and the design tokens, and a change to either has to be made twice.
