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

## Comments

### Frontend address: Vercel alias, not verified live

The frontend is aliased to `https://cardstack-pr-<N>.vercel.app` through Vercel's REST API (`POST /v2/deployments/{id}/aliases`), right after `vercel deploy`. The manifest's derive entry for `APP_CLIENT_URLS` is that origin and is no longer a placeholder: the workflow's `FRONTEND_HOST` must match it, and provisioning fails before writing anything when it does not. Not run live (no Vercel credentials), so whether Vercel accepts a free `*.vercel.app` alias for a CLI deployment is unconfirmed; the step fails loudly if not. Fallback if ticket 05's live run shows it does not work: Vercel's per-branch address (`<project>-git-<branch>-<team>.vercel.app`), which needs the branch slug rules reproduced in the workflow.

### Removals without a deployment: `variableCollectionUpsert` with `replace` and `skipDeploys`

Checked against the pinned `@railway/cli@5.49.3` source and Railway's GraphQL schema (`src/gql/schema.json` of that tag):

- `variableDelete` (input: environment, name, project, service) has no `skipDeploys`.
- `railway variable edit --yes --skip-deploys --confirm-destructive` does not help: its apply is one `variableCollectionUpsert` with `skipDeploys` for the sets, then one `variableDelete` per removal without it, so each removal deploys.
- `variableCollectionUpsert` takes `skipDeploys` and `replace` ("removes all existing variables before upserting the new collection"), so one call writes the whole variable set with no deployment.

The workflow lists the variables (`railway variable list --json`), runs the ticket 01 script, and sends `listing minus RAILWAY_*, plus sets, minus removals` with `replace: true, skipDeploys: true`. Guards, so the replace cannot lose anything: a sealed variable (listed with a null value) that the manifest does not remove fails the run, because it cannot be read back; `RAILWAY_*` variables are left out of the payload. A step with nothing to change makes no call. Every plan value of 8 characters or more (and the values of the variables being removed) is masked with `::add-mask::` before anything else runs, and the plan JSON is never printed. PyYAML is checked and installed with apt when missing.

Not run live (no Railway credentials). Ticket 05 must confirm on a live PR environment: one API deployment per push, and that `variable list --json` returns raw values (references such as `${{shared.X}}` stay references, not rendered text), since `replace` writes back what the listing returned. If either fails, the alternative is staging the changes with `railway environment edit --stage` and committing with `environmentPatchCommitStaged(skipDeploys: true)`.
