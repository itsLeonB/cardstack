# 07: The preview environment always has a running API

**What to build:** A preview environment that can sign in and reach its database even when the pull request never touched `backend/`. Focused PR Environments stay on, so `watchPatterns: ["/backend/**"]` in `backend/infra/.railway/railway.ts` still skips the api service on a frontend-only push. The provisioning workflow covers that case: when the PR's Railway environment has no api deployment to speak of, it creates one, once, and then leaves the environment alone for the rest of the PR's life.

**Blocked by:** 03 (the variables the API boots with), 05 (live run).

**Status:** ready-for-agent

- [ ] A pull request that changes only `frontend/` gets a working API: after its first deployment, `GET <api domain>/health` answers, on the Neon branch and with `APP_ENV=preview`.
- [ ] The trigger fires only when the environment has no api deployment other than `SKIPPED`, `FAILED` or `CRASHED` records, and it happens after the variables are written, so the API boots already configured.
- [ ] A `QUEUED`, `BUILDING`, `DEPLOYING`, `SUCCESS` or sleeping deployment stops the trigger: nothing is created on a later push, and Railway's own watch-path deploy still covers a push that does change `backend/`.
- [ ] A `FAILED` or `CRASHED` deployment is retried at the next push, at most twice per environment; the third failure stops the workflow from deploying anything and the PR comment says so.
- [ ] The triggered deployment asks for the pull request's head commit, not for whatever commit the service is associated with.
- [ ] No step waits for the triggered deployment, so Railway's Wait for CI cannot deadlock against the run that created it.
- [ ] The PR comment states the deploy rule accurately and reports an exhausted retry budget.
- [ ] The workflow header comment and `docs/agents/deployment.md` describe the trigger, its state check and its retry budget.
- [ ] Verified on a live frontend-only pull request: a skipped service leaves a `SKIPPED` deployment record (confirmed, see the comment below) and `serviceInstanceDeployV2` is not itself filtered by watch patterns (not yet confirmed).
- [ ] `yamllint` passes.

## Comments

### The skipping is real, and it leaves a record

Pull request #72 (`feat(frontend): add to wishlist from the catalog`, merged 2026-11-02) touched only `.scratch/`, `compose.yaml` and `frontend/`. Railway's comment on it lists one api row: status `Skipped`, with a deployment id, and a live `api-cardstack-pr-72.up.railway.app` domain beside it. So a skipped service is not a vanished service: the environment exists, the domain is allocated, and the deployment record exists in a `SKIPPED` state. Any check that only asks "is there a deployment?" answers yes and never deploys. The check has to read the status.

The same record shape also means the status check cannot look at the *latest* deployment alone. A frontend-only PR that has already run its API records a fresh `SKIPPED` on every later push, so "latest is `SKIPPED`" means "deploy" again, which is the loop this ticket exists to prevent. The question is "has this environment ever had an api deployment that was not skipped", asked across the environment's deployments.

### Why the workflow deploys at all, and not Railway

Focused PR Environments is a project setting (Project Settings → Environments → PR Environments) that makes Railway use watch paths to decide which services a PR environment deploys. Turning it off would give every PR environment a full stack copy with no code change, at the cost of an api build on every frontend-only push. Tickets 03 and 05 fixed the one-deployment-per-push cost model with `skipDeploys: true`; this ticket keeps that and pays the build only when the environment has no API.

### How the run decides

- **Signal: Railway's state, not the diff.** The workflow reads the api service's deployments in the PR environment instead of parsing the pull request's file list. A diff check would copy `watchPatterns` out of `railway.ts` and drift silently when either side changes; the state check cannot disagree with Railway. The pull request's file list is also a paginated API call, and `github.event.pull_request.changed_files` gives only a count.
- **No waiting, no polling.** The deployment this run creates is itself held by Railway's Wait for CI until every workflow on the commit finishes, this one included, so polling it to health would wait on itself. A `QUEUED`/`BUILDING`/`DEPLOYING` record therefore means "do nothing this run"; the deployment finishes on its own and the next push re-decides.
- **The budget lives in the deployment records.** Retrying while the environment holds fewer than two `FAILED`/`CRASHED` api deployments needs no new storage, survives pushes, and resets when Railway recreates the environment on a reopened PR. A counter in the PR comment would make a hand-edited comment load-bearing; a counter in a Railway variable would have to join `preview-vars.yml`, or every run fails the manifest's unknown-variable check.
- **The budget counts Railway's failures too.** `serviceInstanceDeployV2` takes no label, so a deployment cannot be marked as the workflow's own. Two broken backend pushes exhaust the budget just as two broken workflow deploys would, and the workflow then stops spending build minutes. A per-commit budget would be unbounded in the case that matters.
- **Which commit: the head SHA.** `serviceInstanceDeployV2(serviceId, environmentId, commitSha)` is called with the pull request's head SHA. Without a `commitSha` it deploys "the commit currently associated with the service", which is undefined for a service Railway skipped.
- **Where: inside the existing variable-apply step.** That step already resolves the `api` service id through the project's GraphQL query, so the trigger costs no second lookup. It must sit after the variable upsert and outside the "already configured" early return, or a pull request whose variables happen to match would skip the deploy decision entirely — and a deploy fired mid-write would boot on production's `APP_ENV` and be refused by the guard from ticket 04.

### Still open

Whether `serviceInstanceDeployV2` is filtered by watch patterns is unconfirmed by Railway's documentation. A live frontend-only pull request answers it: if the triggered deploy is skipped too, the fallback is the diff check above, or turning Focused PR Environments off and accepting the extra builds.
