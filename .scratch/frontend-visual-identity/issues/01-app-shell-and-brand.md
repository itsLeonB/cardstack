# 01: App shell and brand

**What to build:** Every page gets a consistent shell: a sticky header with navigation that adapts to guest vs signed-in, a user menu, a theme toggle, a minimal footer, and a real brand (wordmark, favicon, manifest, title). Existing pages render inside it unchanged. This is the prefactor everything else builds on. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** None (can start immediately)

**Status:** done — implemented (frontend `4da4449` + `1ae0623`, `ad4b716`, `1eac445`), merged to `main` via PR #17.

- [ ] Header, `main` landmark and footer wrap every route, with a skip-to-content link as the first focusable element and visible focus states
- [ ] Guests see Catalog, Log in and Register (Register as the primary action); signed-in users see Catalog, Collections, Master Inventory and a user menu (email, Account, Log out). The label is "Master Inventory", never "Inventory"
- [ ] The current section is marked as the current page for assistive tech and styled distinctly
- [ ] Mobile-first: on small screens the nav collapses into a menu that opens and closes by tap and keyboard; no horizontal scroll at phone width on any existing page
- [ ] Header nav is plain links inside a `nav` with no list markup, so existing Playwright specs that count list items still pass unmodified
- [ ] Footer: wordmark, one-line description, an MVP note (no nav links; the header already carries them) (personal MVP, Indonesian print editions). No legal links
- [ ] Theme toggle (light, dark, system), default system, persisted, applied before first paint with no flash. Prefer a proven solution from the TanStack ecosystem or the shadcn-documented approach over a hand-rolled one; confirm against current docs first
- [ ] Wordmark (stacked-cards glyph plus "Cardstack" in the heading font), SVG favicon, corrected manifest (no references to missing icons), real document title, `lang`, description and theme colour in the root head
- [ ] Playwright: guest header and signed-in header, navigation between main areas, theme persistence, and an axe check on the shell
- [ ] RTL (network mocked): header renders the right links for guest vs signed-in
- [ ] Any shell UI not verified in a browser is called out as such; `bun run lint`, typecheck, tests and build pass
