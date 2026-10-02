# 03: Landing and dashboard at `/`, plus Account

**What to build:** The home page explains the product to guests and is useful to signed-in users, at the same URL. While the session resolves, a skeleton shows (never marketing copy), then the guest landing or the dashboard. Account becomes a settings-style page. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 02

**Status:** done

- [x] Guest landing: hero with value statement and tagline, two actions (Create account, Browse the catalog), a three-step "how it works" (browse the Catalog, add Cards to Collections, see your Master Inventory), and a closing call to action. No fabricated screenshots or testimonials, no live catalog embed
- [x] Signed-in dashboard: greeting, the user's Collections with a "New collection" action, a count labelled "distinct cards" from the Master Inventory list total, and quick actions. A signed-in user with no Collections sees guidance on what to do first
- [x] Before the label ships, verify against the backend what the Master Inventory total counts (distinct cards vs entries) and word the label to match what it truly means
- [x] Skeleton while the session resolves; a failed or unauthenticated session check renders the guest landing, with no flash of the landing for a signed-in user
- [x] Account page is settings-style, reached from the user menu, and no longer carries navigation links; Log out still works
- [x] Total owned quantity and per-Collection card counts are not added (out of scope; separate ticket 19 covers collection counts)
- [x] Check TanStack Start's prerender option against SPA mode in current docs; adopt prerendering of the landing only if compatible, otherwise record that it is not used
- [x] Playwright: guest at `/` sees the landing; signed-in at `/` sees the dashboard; axe check on landing and dashboard
- [x] RTL (network mocked): skeleton during session load, failed session check
- [x] Mobile-first at phone width; lint, typecheck, tests and build pass; note anything not verified in a browser
