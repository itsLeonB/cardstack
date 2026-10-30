# 08: Browser suite against the built stack

**What to build:** the existing Playwright suite runs from the frontend tooling container against the built user app and the API in the stack, with the seeded fixture, so the browser tests exercise the production build.

**Blocked by:** 06 (User app image, serve script and frontend verification).

**Delivery:** PR 2, branch `chore/compose-stack`, orchestrator-owned (script, workflow, compose), with a frontend worker only if the Playwright configuration needs a change.

**Status:** ready-for-agent

- [ ] The end-to-end step of the frontend verification script seeds the database, rebuilds any changed app image, and runs the suite from the tooling container with the base URL pointing at the user app service. It does not start a development server.
- [ ] The suite's guest specs pass against the built app in the stack with no secrets.
- [ ] The signed-in specs run when the Clerk variables are present in the existing env files and skip with a stated reason when absent, as today. Whether the Clerk development instance accepts the in-network origin is verified; if it refuses, signed-in specs are documented as not runnable in the stack and CI keeps its current path for them, with the guest specs still running in the stack.
- [ ] The end-to-end CI workflow runs the same script step, keeping its job name, with the Playwright report still uploaded on failure.
- [ ] Any difference the built app surfaces compared with the development server is recorded in the ticket's comments, with a fix or a follow-up ticket for each.
- [ ] The YAML passes the repo's YAML linter.
