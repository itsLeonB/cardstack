# 02: Page layout system and breadcrumbs

**What to build:** One shared way to lay out every page. A page container with three widths and a page header, adopted by every existing page, with breadcrumbs replacing every "← Back" link on nested pages. Content and behavior of catalog, collections and inventory pages are otherwise unchanged. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 01

**Status:** done

- [x] Page container with `narrow` (auth), `default` and `wide` variants, and a page header with title, optional description and actions slot, used by all existing pages in place of per-page wrappers
- [x] `wide` is used only for catalog search and Master Inventory
- [x] Breadcrumbs appear on nested pages only (for example Catalog > Set > Card, Collections > Collection > Edit) and are an accessible breadcrumb navigation; top-level pages (Catalog, Collections, Master Inventory) have none
- [x] Every ad-hoc "← Back" link is removed
- [x] Pages remain usable at phone width with no horizontal scroll
- [x] Existing Playwright specs (catalog browse and search) and unit tests pass without modification; list-item counts and heading text they assert are unchanged
- [x] New Playwright check that a nested page shows its breadcrumb trail and that following a crumb navigates up
- [x] The layout primitives and their usage rule are recorded in the frontend conventions doc
- [x] Lint, typecheck, tests and build pass; note anything not verified in a browser
