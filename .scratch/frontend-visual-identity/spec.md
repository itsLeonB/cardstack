# Frontend visual identity

Status: ready-for-agent

## Problem Statement

Cardstack works, but it does not look or feel like an app. The home page is the untouched TanStack Start starter, with a placeholder title and favicon. No page has a header, footer or navigation, so a visitor cannot move between the Catalog, Collections, Master Inventory and Account without typing URLs or following ad-hoc text links. Each page invents its own layout and its own "back" link, so the experience is inconsistent. A first-time visitor cannot tell what the product does or how to start, and a returning user sees the same starter page as a stranger. Not-found and crash states are bare or missing, there is no dark mode although the dark tokens exist, and there is no feedback when logout fails.

## Solution

Give the app a coherent, mobile-first visual identity and one shared way to lay out every page.

- A consistent shell: sticky header with navigation that adapts to guest vs signed-in, a user menu, a theme toggle, and a minimal footer.
- A single generalized page layout (container widths, page header, breadcrumbs on nested pages) that existing pages are fitted into, replacing per-page wrappers and "back" links.
- At `/`, a guest sees a landing page that explains what Cardstack does and how to start, and a signed-in user sees a small dashboard. Both live at the same URL.
- Polished login and register pages, a proper not-found page, a minimal crash fallback, and toast feedback for shell-level actions.
- A real brand: wordmark, favicon, manifest, title, and site-wide metadata for search and link previews.
- Accessibility and responsiveness are requirements, not polish: keyboard navigation, a skip link, visible focus, correct landmarks, and no horizontal scroll at phone width.

## User Stories

1. As a first-time visitor, I want the home page to say what Cardstack is and who it is for, so that I can decide in seconds whether it is useful to me.
2. As a first-time visitor, I want a clear "how it works" explanation (browse the catalog, add cards to Collections, see your Master Inventory), so that I understand how I would use the app.
3. As a first-time visitor, I want prominent "Create account" and "Browse the catalog" actions on the landing page, so that I can start immediately or look around first.
4. As a guest, I want to browse the public Catalog without signing up, so that I can evaluate the app before committing.
5. As a guest, I want Log in and Register always visible in the header, so that I never have to hunt for them.
6. As a signed-in user, I want `/` to show my dashboard instead of marketing copy, so that the home page is useful to me.
7. As a signed-in user, I want a greeting and my Collections listed on the dashboard, so that I can jump straight into the one I want.
8. As a signed-in user, I want to see how many distinct cards I own on the dashboard, so that I get a quick sense of my Master Inventory.
9. As a signed-in user, I want a "New collection" action on the dashboard, so that I can start a new binder, box or deck quickly.
10. As a signed-in user with no Collections yet, I want the dashboard to explain what to do first, so that I am not looking at an empty screen.
11. As a signed-in user, I want a header with Catalog, Collections and Master Inventory, so that I can reach every main area from any page.
12. As a signed-in user, I want a user menu with my email, Account and Log out, so that account actions are in a predictable place.
13. As a user, I want the current section highlighted in the header, so that I always know where I am.
14. As a user on a nested page, I want breadcrumbs instead of a one-off back button, so that I can see my place in the hierarchy and go up a level.
15. As a user on a top-level page, I want no redundant back link, so that the header navigation is the single way around.
16. As a user, I want every page to share consistent widths, spacing and page headings, so that the app feels like one product.
17. As a user working with filters on search and Master Inventory, I want those pages to use the wider layout, so that results have room.
18. As a mobile user, I want the header to collapse into a menu that opens and closes with a tap, so that navigation fits a small screen.
19. As a mobile user, I want every page to work at phone width without horizontal scrolling, so that the app is usable one-handed.
20. As a user, I want to choose light, dark or system theme from the header, so that the app is comfortable in my environment.
21. As a user, I want my theme choice remembered across visits, so that I do not have to set it again.
22. As a user with a dark system theme, I want the page not to flash light on load, so that my eyes are not hit by a white screen.
23. As a keyboard user, I want a skip-to-content link and visible focus states, so that I can navigate without a mouse.
24. As a screen-reader user, I want proper landmarks, heading order and labelled controls, so that I can understand each page.
25. As a screen-reader user, I want the current page marked as current in navigation and form errors announced and tied to their fields, so that I can act on them.
26. As a user, I want the register page to explain what I get and link to login, so that I can switch easily.
27. As a user, I want the login page to link to register, so that I can create an account if I do not have one.
28. As a user filling in a password, I want a show/hide toggle, so that I can check what I typed.
29. As a user, I want my browser or password manager to autofill email and password correctly, so that signing in is quick.
30. As a user, I want field-level error messages next to the input that caused them, so that I know what to fix.
31. As a signed-in user visiting login or register, I want to be sent to the home page instead, so that I am not shown a form I do not need. (Absorbs existing ticket 18.)
32. As a user redirected to login from a protected page, I want to land back where I was after signing in, so that I do not lose my place. (Currently broken: it falls back to Account. Fixed here; absorbs existing ticket 17.)
33. As a user who opens a URL that does not exist, I want a clear "page not found" page inside the normal layout, so that I can find my way back.
34. As a user who opens a Collection or Card that does not exist, I want a message that says that specific thing was not found, so that I am not confused by a generic error.
35. As a user, I want an unexpected crash to show a friendly fallback with a Reload button, so that I am not left on a blank screen.
36. As a user, I never want to see a stack trace or raw server message in the UI, so that the app looks trustworthy and leaks nothing.
37. As a user, I want a toast telling me when logout fails, so that I know my session is still active.
38. As a user, I want a short notice when I am sent to login because my session ended, so that I understand why. (Delivered by existing ticket 31, not this spec.)
39. As a user, I want data-load failures on a page to show an inline message with a Retry action, so that a temporary API problem does not strand me.
40. As a returning signed-in user, I want `/` to show a loading skeleton rather than flashing the marketing page while my session loads, so that the page does not jump.
41. As a visitor arriving from a search engine, I want a descriptive page title and description for the landing and catalog pages, so that I know what I am clicking.
42. As someone sharing a link to Cardstack, I want a recognizable preview (title, description, image), so that the link looks credible.
43. As a user, I want a real favicon, app name and theme colour on my browser tab and home screen, so that the app is recognizable.
44. As a user, I want a footer that says what Cardstack is and links to the main areas, so that every page has a clear end and useful exits.
45. As the app owner, I want authenticated pages and the auth forms excluded from search indexing, so that only public pages appear in results.
46. As the app owner, I want a sitemap listing the landing page and the Catalog, so that search engines find the public entry points.
47. As the app owner, I want the visual identity covered by smoke and accessibility tests, so that future changes do not silently break navigation or accessibility.

## Implementation Decisions

**Brand and assets**
- Keep the existing palette (amber primary on olive neutrals) and fonts (Figtree body, Instrument Sans headings). No new colour direction.
- Add a wordmark: a stacked-cards glyph plus "Cardstack" in the heading font. Provide an SVG favicon and replace the starter manifest, which currently names a different app and references icons that do not exist.
- Tagline: "Track every card you own, across every binder."
- Replace the placeholder document title; set `lang`, description and theme colour in the root head.

**App shell and layout system**
- One shell wraps every route: skip link, sticky header, a single `main` landmark, footer.
- Header: guests see Catalog, Log in and Register (as the primary action). Signed-in users see Catalog, Collections, Master Inventory and a user menu (email, Account, Log out). The UI label is "Master Inventory", per the glossary, never plain "Inventory". Catalog stays visible to guests because those routes are public.
- Mobile-first: the header collapses to a menu on small screens. Navigation is plain links inside a `nav` element, with no list markup, so existing end-to-end tests that count list items keep working. The current section is marked with the current-page attribute.
- Footer: wordmark, one-line description, links that mirror the header per auth state, and a note that this is a personal MVP covering Indonesian print editions. No legal links until real policies exist.
- Page layout primitives, extracted now because reuse is real across eight or more pages: a page container with three width variants (`narrow` for auth, `default` for most pages, `wide` for catalog search and Master Inventory only), and a page header with title, optional description and an actions slot.
- Breadcrumbs replace every per-page "back" link, on nested pages only (for example Catalog > Set > Card, Collections > Collection > Edit). Top-level pages (Catalog, Collections, Master Inventory) get none.
- Existing catalog, collection and inventory pages are only fitted into this layout. Their content and behavior are otherwise unchanged.

**Theme**
- Light, dark and system, defaulting to system, chosen from the header, persisted, applied before first paint to avoid a flash. Prefer a proven, widely used solution (TanStack ecosystem first, then the shadcn-documented approach) over a hand-rolled provider; confirm against current documentation before building.

**Routing and the `/` page**
- The app runs TanStack Start in SPA mode: the server returns one shell and the browser renders everything. There is no server rendering of route content, and the server cannot read the session, which is held in HttpOnly cookies and resolved by calling the backend from the browser.
- `/` is a single URL for everyone. While the session resolves it shows a skeleton (never marketing copy), then renders the guest landing or the signed-in dashboard. Search crawlers carry no session, so they see the guest landing. Whether prerendering the landing at build time is compatible with SPA mode is to be confirmed in the TanStack Start documentation; adopt it only if it is.
- Guest landing: hero with value statement and two actions (create account, browse catalog), a three-step "how it works" (browse the Catalog, add Cards to Collections, see your Master Inventory), and a closing call to action. No fabricated screenshots, testimonials or live catalog embed.
- Signed-in dashboard (small): greeting, the user's Collections with a "New collection" action, a count labelled "distinct cards" taken from the Master Inventory list's total, and quick actions. Total owned quantity and per-Collection card counts are not available from the API today and are out of scope. Before shipping the "distinct cards" label, verify against the backend what the Master Inventory total actually counts.
- The Account page becomes a settings-style page reached from the user menu and no longer carries navigation links.

**Auth pages**
- Centered card with brand mark, link between login and register, correct autocomplete hints for email and passwords, field errors tied to their inputs and announced, a pending state on submit, a show/hide password toggle. The existing "account created" notice is preserved. No social login and no forgot-password flow, per ADR-0004.
- Absorbs two existing tickets (17 and 18 in the MVP issue folder, now marked absorbed): signed-in users visiting login or register are redirected to `/`, via a guest guard that mirrors the existing auth guard; and the post-login redirect is fixed so a user sent to login from a protected page lands back there (the guard currently passes an absolute URL that the login page's same-origin check rejects). Keep the same-origin safety check.

**Errors and feedback**
- Add a root not-found page and a minimal root crash fallback with a Reload action, both rendered inside the shell and never showing raw error text. Missing Collections and Cards should resolve to the not-found page with a message naming what was not found.
- Add a toast library and use it for shell-level actions only in this spec: logout, including the currently silent failure. The session-expired notice and token-refresh behavior belong to existing ticket 31, which may land before or after; whichever runs first installs the library and mounts the toaster, the other acknowledges it is already there. Existing inline errors for forms, dialogs, per-card quantity edits and page-load failures stay as they are. Per-feature mutations are not migrated to toasts in this spec.
- Convention to record in the frontend conventions doc: toasts for failures of non-form actions, inline errors for form validation and for failed page data (with a Retry action).

**SEO and metadata (SPA constraints)**
- Static Open Graph tags, theme colour and description live in the root head, with one static preview image for the whole site. Per-route titles are set from the browser. Per-route Open Graph is not attempted because link-preview crawlers do not run scripts.
- Sitemap lists the landing page and the Catalog index only (not every card). Robots file disallows authenticated paths. Authenticated pages, login and register carry a noindex directive.

**Conventions**
- New code conventions introduced (layout primitives, toast usage, theme) are recorded in `docs/agents/conventions/frontend.md`, not in AGENTS.md or agent definitions.

## Testing Decisions

- A good test exercises what a user can see or do (navigation reachable, correct links for the auth state, correct page rendered, accessible name present), not component internals, class names or styling. Visual styling (palette, spacing) is deliberately not asserted and is verified manually in a browser; any UI work not browser-verified must say so.
- Seam 1, primary: Playwright end-to-end against the seeded backend. Cover: guest header and landing; signed-in header and dashboard at `/`; navigation between main areas; login and register flows including already-signed-in redirect; the not-found page; theme toggle persistence. Run an axe accessibility check on the landing, login, register, dashboard, catalog and not-found pages. Add the axe integration if not installed.
- Seam 2: Vitest with React Testing Library and the network mocked (ADR-0005). Cover what end-to-end makes costly: the `/` skeleton while the session loads, a failed or unauthenticated session check, the crash fallback rendering, header behavior per auth state, and the logout-failure toast.
- Prior art: the existing catalog browse and search specs for end-to-end; the colocated component tests for Vitest. Existing specs count list-item roles and assert headings, so shell markup must not add list items or alter those headings.
- No new seams are introduced.

## Out of Scope

- Redesigning the catalog, collections or inventory pages beyond fitting them into the shared layout.
- Dashboard totals that need new backend data: total owned quantity, per-Collection card counts. Filed as a later phase.
- Migrating existing per-feature mutation errors to toasts.
- Server rendering, per-route Open Graph images, and a sitemap of individual cards.
- Token refresh and the session-expired notice (existing ticket 31); collection card counts and clickable cards (19); guest catalog locking (24); catalog set images, ordering and infinite scroll (13, 14, 25); backend auth error sanitizing (30). These stay as separate tickets and may merge-conflict mildly with the layout migration.
- Privacy policy, terms, forgot-password, email verification, social login (ADR-0004).
- New game or region support, wishlist, or any domain-model change. No `CONTEXT.md` or ADR change is needed.

## Further Notes

- Source of decisions: grilling session on 2026-10-01; everything above was confirmed by the user.
- Verification items to resolve during implementation, not decisions: the current TanStack Start prerender option under SPA mode; the proven theme solution for TanStack Start; what the Master Inventory total counts (distinct cards vs entries).
- The existing end-to-end specs depend on list-item counts and heading text; protect them.
