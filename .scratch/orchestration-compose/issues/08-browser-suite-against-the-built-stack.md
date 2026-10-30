# 08: Browser suite against the built stack

**What to build:** the existing Playwright suite runs from the frontend tooling container against the built user app and the API in the stack, with the seeded fixture, so the browser tests exercise the production build.

**Blocked by:** 06 (User app image, serve script and frontend verification).

**Delivery:** PR 2, branch `chore/compose-stack`, orchestrator-owned (script, workflow, compose), with a frontend worker only if the Playwright configuration needs a change.

**Status:** done

- [x] The end-to-end step of the frontend verification script seeds the database, rebuilds any changed app image, and runs the suite from the tooling container with the base URL pointing at the user app service. It does not start a development server.
- [x] The suite's guest specs pass against the built app in the stack with no secrets.
- [x] The signed-in specs run when the Clerk variables are present in the existing env files and skip with a stated reason when absent, as today. Whether the Clerk development instance accepts the in-network origin is verified; if it refuses, signed-in specs are documented as not runnable in the stack and CI keeps its current path for them, with the guest specs still running in the stack.
- [x] The end-to-end CI workflow runs the same script step, keeping its job name, with the Playwright report still uploaded on failure.
- [x] Any difference the built app surfaces compared with the development server is recorded in the ticket's comments, with a fix or a follow-up ticket for each.
- [x] The YAML passes the repo's YAML linter.

## Comments

Differences the built app surfaced compared with the dev server, 2026-10-30:

- **Origin is not a secure context.** `http://web:3000` is not `localhost`, so `crypto.subtle` is missing. Clerk logs "Suffixed cookie failed" and carries on; sign-in works. A spec that needs a secure browser API would need Chromium's `--unsafely-treat-insecure-origin-as-secure`. Documented in `docs/agents/testing.md`. The Clerk development instance accepts `http://web:3000`: clerk-js loads, `dev_browser`, `environment` and `client` answer 200, and the signed-in specs (real sign-in through the form) pass.
- **`auth.spec.ts` hardcoded `localhost`** in the external-redirect assertion. Fixed: it compares against `baseURL`.
- **Real production bug: no `aria-current` on the header links after a direct load.** The SPA shell is prerendered at `/`, and React keeps the attributes it hydrates, so after a direct load of `/catalog/search` the "Catalog" link had none and the Wordmark was marked current on every route. The dev server renders the real URL on the server, so it never showed. Fixed in `site-header.tsx` (the header content remounts once hydrated), with a regression test. Any other component that reads the location during the first render would hit the same mismatch; none found in the shell today.
- **Clerk load time makes the suite flaky from this machine.** clerk-js takes 2 to 9 seconds to render a Clerk page here (the dev server is slower still, 6 to 7 seconds, because it compiles on demand), against the specs' 5-second expect timeout. A local run without retries fails a varying set of `/auth/*` and signed-in specs (4 to 22 of 83); with `CI=true` (2 retries) most pass on retry. Not caused by the stack: the same probe against the dev server is slower. Follow-up: raise the expect timeout, or wait for Clerk to load in a shared fixture.
- **Backend `go test -race` and the user-app `-collection-move.test.tsx`** time out under load on this machine; the same test fails on the host with no container involved.
