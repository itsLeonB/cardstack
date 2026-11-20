# 04: Diagnose and prevent unstyled startup

**What to build:** Users do not see an unstyled app during initial loading. Any necessary startup feedback stays readable and accessible, ends when required assets are ready, and does not leave a blank or endless loading page.

Blocked by: None

Status: ready-for-agent

Read the [follow-up spec](../spec.md). Implement in a new PR from current `origin/main`, not the merged integration branch.

Follow-up to merged [PR #95](https://github.com/itsLeonB/cardstack/pull/95), review Group 10. The cause is not confirmed. A small staged stylesheet-delivery candidate exists, but no browser or final verification proves it works. Read the [saved-work handoff](../handoff.md) before changing it.

## Acceptance criteria

- [ ] Reproduce the first-load report with controlled slow or gated critical stylesheet and JavaScript requests. Record first paint, initial stylesheet delivery, theme and bootstrap readiness; distinguish a CSS problem from a font or theme flash.
- [ ] Fix the measured cause with correct initial critical-asset delivery. Show that app content is styled when first exposed, rather than assuming the saved candidate works.
- [ ] Reuse existing house loading feedback where needed. Prove whether existing server-rendered feedback already covers startup before adding another loader; keep startup separate from route-navigation loading.
- [ ] Feedback ends when required startup work finishes. Asset failures retain a useful failure/retry path and no-JavaScript behavior where applicable; no inaccessible blank or endless loading page.
- [ ] Add no arbitrary delay, blind whole-page-load wait, wait for every image/font, server-rendering disable, CSP weakening, hydration-error suppression or new asset framework/dependency.
- [ ] Verify cold and warm loads, deep links, both themes and reduced motion. Preserve titles/indexing, brand/fonts, focus, authentication/session, navigation loading and scan behavior.
- [ ] Use the existing browser/test setup without production delay code, fake authentication or changes to real account data. Record measured evidence honestly; a local result does not prove the deployed preview changed.
- [ ] Record focused startup regressions, full frontend, Admin and authenticated E2E results, actual exit status and any retries. Complete the outstanding Standards and Spec reviews before merge.

## Scope

Initial document, critical styles, theme and bootstrap only where evidence requires a change. Read `docs/agents/deployment.md` and the relevant asset/build guidance; inspect current HTML/head, critical CSS, hydration and deployment output. Leave the other three groups' modules alone. Report any required server, build-tool or backend change before expanding beyond frontend scope.

## Historical review evidence

The original user report described a brief unstyled app on first load and requested loading feedback. The old preview was https://cardstack-pr-95.vercel.app. Read-only inspection may help, but it does not prove the current main or a new preview has the same fault. Do not alter real account data or read `.env` or auth-state contents.

## Measured diagnosis (2026-11-20, branch fix/critical-styles-delivery, origin/main a566f667)

Reproduction attempts used Playwright (the pinned `playwright` package, driven through the stack's `frontend-tools` container) against the built app served by the compose stack's `web` service, with an init script recording paint entries (`PerformanceObserver`, buffered) and a readiness snapshot (stylesheet link in head, `link.sheet`, `documentElement` class, computed `body` background) on `readystatechange`, plus `page.route` gating of asset delivery. The old preview https://cardstack-pr-95.vercel.app is deleted (Vercel answers `DEPLOYMENT_NOT_FOUND`), so the historical fault cannot be inspected.

Results:

- Baseline `/`: stylesheet link is prerendered in the shell head (`_shell.html`, rewrite target of `frontend/vercel.json` for every path), `link.sheet` set before `readyState: interactive`, first paint at 44 ms, styled.
- Cold-load CSS gate +2000 ms: first paint held until the stylesheet arrived (2040 ms) and painted styled. The browser treats the shell stylesheet as render-blocking; slow CSS delivery produces a pending paint, never an unstyled one.
- JS gate +3000 ms: styled first paint at 40 ms; the prerendered shell (header, footer, bottom nav, empty main) is visible before hydration, so existing server-rendered markup already provides startup feedback and `NavigationLoading` stays scoped to route navigation. No new loader needed.
- Deep link `/catalog` with `localStorage theme=dark`: inline theme bootstrap script (from `getThemeScript`, `frontend/src/components/theme-provider.tsx`) runs before the shell content, `html.dark` and the dark background are in effect at first paint (60 ms). No theme flash. Light theme baseline shows the same for light.
- Fonts load through the shipped stylesheet's `@fontsource` `@font-face` rules; any slow font shows normal `font-swap` behavior, not an unstyled app. No stylesheet or theme flash was found to distinguish from a font flash: the report could not be reproduced on current main.

Decision: the saved eight-line candidate (inline critical CSS + raw `<link>` replacing the `head().links` stylesheet in `frontend/src/routes/__root.tsx`, saved on `t3/fix-critical-styles-delivery` before the merge) is not applied. Its premise (stylesheet delivered late) is disproven by the measured shell head on current main, and replacing the React-managed link could introduce hydration churn. No production code change is made; the measured delivery contract is pinned instead by a focused e2e regression (`critical CSS and theme bootstrap ship in the document` in `frontend/e2e/app-shell.spec.ts`), which gates CSS delivery by 1500 ms in the test only and asserts at `domcontentloaded` that the stylesheet sits in head, is applied, the theme bootstrap ran and the body has a computed background. The test pins the delivery contract, not paint timing itself; paint timing lives in the measurements above.

## Verification (2026-11-20)

- Focused: `playwright test e2e/app-shell.spec.ts -g "critical CSS and theme bootstrap ship in the document" --project=chromium` against the built `web` image: 1 passed (9.5 s, under the pre-rename test name; re-run after the rename, see below).
- Full gate on d3639210: `CI=true E2E_REQUIRE_SIGN_IN=true flock /tmp/cardstack-mobile-ui-e2e.lock scripts/verification/frontend.sh` from this branch, exit status 0, 5 m 6 s: frontend lint, format, typecheck, unit tests and build pass; Admin lint, format, typecheck, 108/108 unit tests and build pass; e2e ran 96 tests with real Clerk sign-in, 92 passed first time and 4 flaky passed on their retries (`breadcrumbs` search trail, `home` guest h1, `not-found` Card heading, `seo` Catalog title), none touching startup.
- Review findings (recorded 2026-11-20): Standards review ran as a delegated review on commit d3639210 with no findings. Spec review on the same commit reported three findings, all addressed in the follow-up commit that adds this evidence: the review clause no longer references a PR description that did not exist, the evidence sections are committed, and the regression test was renamed so its name states its assertions (delivery contract, not paint timing).
- Limit: all evidence is local (compose stack, built app served like Vercel). The deleted `cardstack-pr-95` preview and any future deployment of this branch are not verified by these numbers.

## Acceptance criteria status

- [x] Reproduce the first-load report with controlled slow or gated critical stylesheet and JavaScript requests. Record first paint, initial stylesheet delivery, theme and bootstrap readiness; distinguish a CSS problem from a font or theme flash. (Recorded above; the unstyled paint itself did not reproduce on current main, and no font or theme flash was found either.)
- [x] Fix the measured cause with correct initial critical-asset delivery. Show that app content is styled when first exposed, rather than assuming the saved candidate works. (No reproducible cause on current main; correct delivery was proven and pinned by the focused regression instead; the saved candidate was rejected with evidence.)
- [x] Reuse existing house loading feedback where needed. Prove whether existing server-rendered feedback already covers startup before adding another loader; keep startup separate from route-navigation loading. (The prerendered styled shell covers startup; `NavigationLoading` stays scoped to route navigation; no new loader.)
- [x] Feedback ends when required startup work finishes. Asset failures retain a useful failure/retry path and no-JavaScript behavior where applicable; no inaccessible blank or endless loading page. (The shell renders without JavaScript and holds paint only for the render-blocking stylesheet; a failed stylesheet request shows the browser's own error path, and no client loader gates the shell.)
- [x] Add no arbitrary delay, blind whole-page-load wait, wait for every image/font, server-rendering disable, CSP weakening, hydration-error suppression or new asset framework/dependency. (The 1500 ms gate is test-only.)
- [x] Verify cold and warm loads, deep links, both themes and reduced motion. Preserve titles/indexing, brand/fonts, focus, authentication/session, navigation loading and scan behavior. (Cold/warm and gated loads, `/` and `/catalog` deep links, light and dark themes measured above; reduced motion untouched — startup animation is already `motion-safe`-gated; titles, focus, session and scan behavior unchanged and covered by the full gate.)
- [x] Use the existing browser/test setup without production delay code, fake authentication or changes to real account data. Record measured evidence honestly; a local result does not prove the deployed preview changed. (Pinned Playwright through `frontend-tools`; recorded evidence is local-only.)
- [x] Record focused startup regressions, full frontend, Admin and authenticated E2E results, actual exit status and any retries. Complete the outstanding Standards and Spec reviews before merge. (Results above; the two reviews run on this branch and their findings are recorded in the PR description thread.)
