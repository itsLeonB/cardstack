# 10: Initial asset loading and unstyled flash

Type: task

Status: claimed

Blocked by: None

Execution: approved by the user's second PR95 review request. Use the existing `feat/mobile-ui-ux` integration branch; do not push from a worker.

## User report

On first app load, the user sees a brief unstyled app before its assets finish downloading. They want appropriate loading feedback instead of unstyled content.

## Investigation and minimal fix

Reproduce an uncached/hard initial navigation under controlled slow critical assets. Record first-paint, stylesheet/bootstrap/theme readiness, and what the user actually sees; distinguish missing/late CSS from theme or font flash before choosing a fix. Inspect the existing HTML/head/critical-CSS and hydration setup and current deployment output. Keep the existing house loading primitives and brand.

Prevent unstyled first paint with correct initial critical-asset delivery and add a small accessible startup loading state where genuinely needed. Do not hide the whole app until every image/font loads, wait on `window.load` blindly, introduce arbitrary delays, disable server rendering/SEO, or build a new asset framework. Avoid blank/forever-loading screens: preserve a usable failure/retry/no-JS path where applicable. Do not weaken CSP, expose secrets, silence hydration errors, or add dependencies. Keep existing navigation loading separate and preserve auth/session, focus, route titles, themes, reduced motion, and scan behavior.

## Boundaries

Own root document/head/bootstrap, ThemeProvider or global startup styles only if the measured cause needs them, and focused startup tests. Group 2 owns catalog/dropdown primitives; Group 3 owns Trades; Group 1 owns Wishlist and binder state. Do not rewrite their modules. Read `docs/agents/deployment.md` and follow only relevant asset/build guidance; frontend-only scope. Report any required root tooling/server/backend change before editing outside scope.

## Proof

Use the existing compose app and Playwright with gated/delayed stylesheet and JavaScript requests, no production delay code. Show that the app is styled when first exposed, feedback lasts only until required assets are ready, and failures do not leave an inaccessible blank page. Cover cold and warm loads, deep links, both themes, and reduced motion. Preserve existing app-shell, SEO, and auth tests. A local browser proof is not a claim that the deployed preview has changed before a push.

Preview origin: https://cardstack-pr-95.vercel.app. Read-only browser inspection is permitted for reproduction, not changes to the user's account. No direct `.env` or auth-state reads. Report unreproduced findings and captured evidence honestly instead of inventing a cause.
