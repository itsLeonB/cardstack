# 06: Metadata, SEO files and indexing rules

**What to build:** Make the app discoverable and shareable, within SPA-mode limits. The server returns one shell and the browser renders everything, so link-preview crawlers see only the static root head; per-route Open Graph is therefore not attempted. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 03, 04, 05

**Status:** ready-for-agent

- [x] Static Open Graph tags, theme colour and description in the root head, with one static preview image for the whole site
- [x] Every route sets a descriptive per-route document title from the browser
- [x] Authenticated pages, login and register carry a noindex directive; the landing and the public Catalog do not
- [x] A sitemap lists the landing page and the Catalog index only; a robots file disallows authenticated paths and points at the sitemap
- [x] Playwright checks the title and noindex presence on representative public and authenticated routes
- [x] Because guest catalog locking (ticket 24) may later change what is public, note in the sitemap decision that it assumes the Catalog stays public today
- [x] Lint, typecheck, tests and build pass; note anything not verified in a browser
