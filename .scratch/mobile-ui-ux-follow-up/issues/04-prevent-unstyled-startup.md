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
