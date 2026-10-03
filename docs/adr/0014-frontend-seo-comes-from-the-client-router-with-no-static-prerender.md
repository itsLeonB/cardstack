# Frontend page titles and noindex come from the client router, with no static prerender of the landing

The frontend runs in TanStack Start's SPA mode, which prerenders only the root route to `_shell.html` by crawling `/`, so that shell carries the landing's title and no route content. Decided to keep it that way: every route sets its title with `pageHead` (`lib/site.ts`) and the `_authenticated` and `auth` layouts add `NOINDEX_META` from the client router, and only the root sets the Open Graph tags, which are static because link-preview crawlers don't run scripts. A crawler that doesn't run scripts sees the landing's title and no noindex on every URL; crawlers that do run scripts get the right title and directives per route.

## Considered Options

- Statically prerender `/` (TanStack Start's top-level `prerender` option combined with `spa: { enabled: true }`). Rejected: the session lives in HttpOnly cookies the server can't read, so a prerendered `/` would have to pick the landing (flashing marketing copy at signed-in users) or the loading skeleton (no better for crawlers than the shell). Revisit only if the home page stops depending on the session.
- Rely on the `<meta name="robots">` tag alone to keep `/auth` out of the index. Rejected: it only exists once scripts run, so `vercel.json` also sends `X-Robots-Tag: noindex` for `/auth/:path*`, the one server-visible directive. Authenticated paths rely on the `robots.txt` disallow instead, which deliberately does not cover `/auth` so the noindex tag stays readable.

## Consequences

New routes under `_authenticated` or `auth` inherit noindex and need no extra work; public routes add nothing. The sitemap and the robots `Sitemap:` line are generated at build time from `VITE_SITE_URL` (`frontend/tools/seo-files.ts`, see `docs/agents/deployment.md`) and list `/` and `/catalog` only, assuming the Catalog stays public. Playwright checks of a layout's child routes reach them by client navigation, because the dev server renders a direct load on the server, where browser-level API stubs don't apply.
