# 06: Metadata, SEO files and indexing rules

**What to build:** Make the app discoverable and shareable, within SPA-mode limits. The server returns one shell and the browser renders everything, so link-preview crawlers see only the static root head; per-route Open Graph is therefore not attempted. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 03, 04, 05

**Status:** done

- [x] Static Open Graph tags, theme colour and description in the root head, with one static preview image for the whole site
- [x] Every route sets a descriptive per-route document title from the browser
- [x] Authenticated pages, login and register carry a noindex directive; the landing and the public Catalog do not
- [x] A sitemap lists the landing page and the Catalog index only; a robots file disallows authenticated paths and points at the sitemap
- [x] Playwright checks the title and noindex presence on representative public and authenticated routes
- [x] Because guest catalog locking (ticket 24) may later change what is public, note in the sitemap decision that it assumes the Catalog stays public today
- [x] Lint, typecheck, tests and build pass; note anything not verified in a browser

## Notes

- Sitemap decision: it lists `/` and the Catalog index only and assumes the Catalog stays public today. If guest catalog locking (ticket 24) changes that, remove `/catalog` from `PUBLIC_PATHS` in `frontend/tools/seo-files.ts` and disallow it in `frontend/public/robots.txt`.
- Set `VITE_SITE_URL` (the production origin, no trailing slash) in the Vercel production environment; the production build now fails without it. Previews skip the sitemap, the robots `Sitemap:` line and `og:image` (with a warning) unless it is set.
- Not verified in a real browser: titles, noindex and the sitemap were checked with headless Chromium against the dev server and a static serve of the production build; the OG image was only viewed as a PNG, not in an actual link-preview crawler.
