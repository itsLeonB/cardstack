# 06: User app image, serve script and frontend verification

**What to build:** the user app runs in the stack from a production-style image served the way Vercel serves it, and an agent or CI runs the whole frontend check through a tooling container with one command.

**Blocked by:** 04 (The stack boots from production images).

**Delivery:** PR 2, branch `chore/compose-stack`. A frontend worker writes the app image, the serve script and the tooling image; the orchestrator writes the compose, script and workflow changes.

**Status:** ready-for-agent

- [ ] The user app has a production-style image kept with the frontend component: a build stage that runs the app's production build with the API address and the Clerk publishable key as build arguments, and a runtime stage that serves the built client output. In the stack the image bakes in the in-network API address.
- [ ] A small Bun script serves the built output. It reads the app's own `vercel.json` and applies its rewrites and headers with URL patterns, serves a real static file when one exists, and falls back to the shell page otherwise.
- [ ] The serve script has a request-in, response-out unit test run against the real `vercel.json`. It covers the fallback to the shell page, the no-index headers on exactly the paths the file names, and a real static file winning over the fallback. It runs under the app's existing unit test command.
- [ ] A frontend tooling image kept with the frontend component is the official Playwright image at the version `package.json` pins, plus bun, with the same bind-mount, host-user and long-lived-process behavior as the backend tooling image. It serves both frontend packages.
- [ ] The frontend verification script accepts a step name (lint, format check, typecheck, unit test, build) and runs all steps in order when given none, for the user app. It starts only the tooling container for steps that need no API.
- [ ] The frontend CI workflow runs the same script per step, keeping its parallel jobs and exact job names, with layer caching and a restored bun package cache.
- [ ] The frontend worker reviews its diff once before reporting. The YAML passes the repo's YAML linter.
