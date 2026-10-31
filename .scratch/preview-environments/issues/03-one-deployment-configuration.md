# 03: One-deployment configuration

**What to build:** Each push configures the PR environment completely before Railway deploys the API, so the API deploys once. The preview frontend gets a stable address derived from the PR number, the API keeps its Railway-provided domain, and the manifest from ticket 01 is applied with no intermediate deployment: preview database and development Clerk settings, the derived allowed origins, `APP_ENV=preview`, and the removal of every production-only variable. The workflow finishes without waiting for the Railway deployment, so Wait for CI and the workflow cannot deadlock. Part of the first pull request.

**Blocked by:** 01 (reconcile script and manifest).

**Status:** ready-for-agent

- [ ] The frontend origin for PR `<N>` is computed before Railway deploys, and the Vercel deployment is aliased to it. If aliasing a CLI deployment does not work, Vercel's per-branch address is used instead and the choice is recorded in this ticket's comments.
- [ ] The API's allowed origins are set to that origin without triggering a deployment, and the API keeps its Railway-provided domain (read before the API deploys, as the workflow already does).
- [ ] The workflow runs the ticket 01 script against the PR environment and applies its sets and removals in one batch with no deployment in between; removals never trigger a deployment.
- [ ] After a provisioning run the PR environment has `APP_ENV=preview`, the preview database settings, the development Clerk settings, and none of the remove-strategy variables.
- [ ] The workflow no longer waits for the Railway deployment or health.
- [ ] The PR comment lists the Neon branch, the API address and the frontend address as soon as provisioning finishes, says the API deploys once checks pass, and is updated in place on later pushes.
- [ ] Closing a PR also removes the Vercel alias, the preview deployments and the branch-scoped frontend variables.
- [ ] Fork and Dependabot PRs are still skipped by provisioning and cleanup.
- [ ] The deployment guide describes the manifest, the one-deployment flow and the PR comment; the workflow header comment matches.
- [ ] The workflow passes `yamllint`.
