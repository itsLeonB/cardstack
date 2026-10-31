# Preview environments: one deploy per push, persistent data, signing in with a copied User

**Status:** ready-for-agent

## Problem Statement

A pull request preview is a stack of three services (a Neon branch, a Railway API and a Vercel frontend), and today it is rebuilt from scratch on every push. The maintainer opening a preview hits four problems.

1. Every push deletes the PR's Neon branch and recreates it from production, so anything created while testing the preview is lost on the next commit.
2. The API deploys more than once per push. It first deploys with the variables Railway inherited, then redeploys when the Neon and Clerk variables are set, then again when the Vercel preview's URL is set as `APP_CLIENT_URLS`, because the Vercel URL only exists after the frontend deploys.
3. A PR environment inherits every variable of production. Until the workflow overrides them, and for any run that is cancelled, skipped or superseded (Railway's Wait for CI ignores those), the PR's API can start with production's database credentials and run the pre-deploy migration against the production database. It also inherits production-only secrets (image bucket write keys, the scan photo bucket, the Admin allowlist, the Gemini key, the edge secret) that a preview must never hold.
4. The preview's database is a copy of production, but its sign-in is the Clerk development instance. The copied User rows carry production Auth Identities, which no development token can match, so the maintainer signs in as a brand new User with no Collections and cannot try the preview against their own data.

## Solution

Each push to a PR produces exactly one API deployment, configured before Railway deploys it. The PR's Neon branch is created once, kept for the life of the PR, and deleted when the PR closes. The preview's frontend origin and API origin are both known before the API deploys, so the API's allowed origins are set once. Every variable a preview API starts with is listed in a checked-in manifest, so nothing from production reaches a preview unless the manifest says so, and an API that starts in a non-production Railway environment without the preview marker refuses to boot. Signing in to a preview with a development Clerk account whose verified email matches a copied User adopts that User, so the maintainer sees their own Collections.

## User Stories

1. As the maintainer, I want a PR's Neon branch created only when it does not exist, so that data I create in the preview survives later pushes
2. As the maintainer, I want a reopened PR to reuse its Neon branch if it still exists and recreate it only when missing, so that closing and reopening does not wipe my test data
3. As the maintainer, I want closing or merging a PR to delete its Neon branch, so that branches do not accumulate
4. As the maintainer, I want to refresh a preview's data by deleting its Neon branch in the console and pushing, so that I can get a fresh copy of production without a dedicated workflow
5. As the maintainer, I want the API to deploy once per push, so that previews are ready sooner and Railway build minutes are not wasted
6. As the maintainer, I want the preview frontend's address to be calculable from the PR number, so that the API's allowed origin is set before the API deploys
7. As the maintainer, I want the preview frontend to keep the same address across pushes, so that my signed-in session and bookmarks keep working
8. As the maintainer, I want the API to keep its Railway-provided domain, so that I do not have to run DNS automation per PR
9. As the maintainer, I want the API's allowed origins set without triggering an intermediate deployment, so that the first deployment already has the final configuration
10. As the maintainer, I want the preview workflow to finish without waiting for the Railway deployment, so that Railway's Wait for CI and the workflow cannot deadlock each other
11. As the maintainer, I want a cancelled or superseded provisioning run to leave the PR environment safe, so that a deployment that Railway lets through still cannot reach production data
12. As the maintainer, I want provisioning runs for different pushes of one PR to queue rather than cancel each other, so that a half-configured environment is not left behind
13. As the maintainer, I want a PR closing to take priority over an in-flight provisioning run, so that cleanup is not blocked behind a deploy nobody will use
14. As the maintainer, I want every variable a PR environment inherits to be classified in a checked-in manifest, so that a new production secret cannot silently leak into previews
15. As the maintainer, I want the workflow to fail when an inherited variable is not in the manifest, so that I am forced to classify it
16. As the maintainer, I want production-only secrets (image bucket write keys, scan photo bucket keys, the Admin allowlist, the Gemini key, the edge secret) removed from every PR environment, so that a preview can neither write to production storage nor expose an Admin surface
17. As the maintainer, I want values that are safe to share (ports, timeouts, pool sizes, telemetry settings) inherited, so that previews behave like production without extra setup
18. As the maintainer, I want the Neon connection settings, the development Clerk settings and the preview marker set in one batch with no deployment between them, so that no deployment ever sees a half-configured set
19. As the maintainer, I want variable removals to cause no extra deployment, so that cleaning inherited variables does not break the one-deployment rule
20. As the maintainer, I want the manifest to be the single source of preview configuration, so that I do not read workflow shell to learn what a preview has
21. As the maintainer, I want an API that starts in a non-production Railway environment without the preview marker to refuse to boot, so that an unprovisioned PR environment fails closed
22. As the maintainer, I want the same refusal in the pre-deploy migration, so that an unprovisioned PR environment cannot migrate the production database
23. As the maintainer, I want production, local development and CI to ignore the guard, so that only PR environments are affected
24. As the maintainer, I want a clear log line when the guard refuses to boot, so that I can tell an unprovisioned environment from a real failure
25. As the maintainer, I want the migration step to run only against the preview database and only on Railway, so that no database credential is needed in the workflow
26. As the maintainer, I want the workflow's own migration step removed, so that one migrator exists and the two cannot disagree
27. As the maintainer, I want a documented fix for a failed preview migration after a rebase or an edited migration, so that I know to delete the Neon branch and push again
28. As the maintainer, I want the PR comment to list the Neon branch, the API and the frontend addresses as soon as provisioning finishes, so that I can open the preview without hunting for URLs
29. As the maintainer, I want the PR comment to say the API deploys once checks pass, so that I am not misled when the API is not up yet
30. As the maintainer, I want the PR comment updated in place on later pushes, so that the PR does not collect duplicate comments
31. As the maintainer, I want closing a PR to remove the preview's frontend alias, its deployments and its branch-scoped frontend variables, so that nothing from a closed PR lingers on Vercel
32. As the maintainer, I want fork and Dependabot PRs skipped, so that provisioning never runs without the repository's secrets
33. As the maintainer, I want a skipped provisioning run to leave the PR environment unable to boot, so that Railway's rule that a skipped check never blocks cannot deploy it against production
34. As the maintainer, I want to sign in to a preview with my development Clerk account and see the Collections of my production User, so that I can test a change against my real data
35. As a developer testing a preview, I want signing in with a verified email that matches no copied User to create a new User as it does today, so that fresh accounts still work
36. As the maintainer, I want adoption to happen only when exactly one User has my verified email, so that an ambiguous match never merges accounts
37. As the maintainer, I want adoption to rewrite the copied User's Auth Identity to my development identity, so that my profile, Collections and Inventory Entries stay attached to one User row
38. As the maintainer, I want concurrent first requests after sign-in to adopt the User once, so that a double page load cannot create a duplicate User or fail
39. As the maintainer, I want adoption to affect only the preview's own Neon branch, so that production's Users are never touched
40. As the maintainer, I want adoption disabled whenever the preview marker is absent, so that it can never run in production or in an unprovisioned PR environment
41. As the maintainer, I want production to keep treating an unknown Auth Identity as a new User, so that nothing about production sign-in changes
42. As the maintainer, I want adoption to rely on the development Clerk instance requiring a verified email at sign-up, so that nobody can adopt a User by typing an address they do not own
43. As a future contributor, I want the preview flow described in the deployment docs, so that I know what a preview creates, what it inherits and how to reset it
44. As a future contributor, I want the Clerk docs to explain adoption and its dependency on verified email, so that I do not turn verification off in the development instance unknowingly

## Implementation Decisions

The work ships as two pull requests. The first covers the preview infrastructure and its lifecycle (stories 1 to 33 and 43). The second covers sign-in adoption (stories 34 to 42 and 44) and lands after the first because it depends on the preview marker the first introduces.

**Preview origins**

- No custom domains and no DNS automation. The API keeps the Railway-provided domain that Railway already assigns to a PR environment, which the workflow can read before the API deploys.
- The frontend is aliased to a free Vercel address derived from the PR number, unclaimed and stable across pushes. The workflow computes it from the PR number before Railway deploys, and uses it both for the API's allowed origins and for the frontend deployment's alias.
- The allowed-origins value is set with the no-redeploy option. Railway's Wait for CI then holds the API's first deployment until the provisioning workflow, like every other workflow on the commit, has finished.
- If aliasing a CLI deployment to the derived address does not work, the fallback is Vercel's per-branch address, which is also derivable from the branch name.
- The workflow does not wait for the Railway deployment. Waiting would deadlock against Wait for CI. The existing wait for the PR environment and for its domain stays, since neither depends on a deployment.

**Neon lifecycle**

- Opening a PR creates the branch from production when it does not exist. A new push reuses it. Reopening reuses it when it still exists and recreates it when it does not. Closing deletes it.
- The create step checks existence first and never deletes. The "delete the stale branch" step and its existence-check-before-delete reasoning go away; the existence check stays, now deciding whether to create.
- There is no refresh workflow. Refreshing is deleting the branch in the Neon console; the next push recreates it.
- The branch is a copy of production data. This is unchanged from today and is out of scope to sanitise.

**Migrations**

- The workflow no longer runs migrations. Railway's pre-deploy command is the only migrator, and it runs on the already-configured preview database. The workflow's Go setup and checkout for that purpose go away, and the Neon password is no longer exposed to a step that runs repository code.
- Because the branch now persists, a rebase onto a migration older than one already applied, or an edit to an applied migration, makes the pre-deploy migration fail. The documented fix is deleting the Neon branch and pushing. No out-of-order migration support is added.

**Preview variable manifest**

- A checked-in manifest classifies every variable the API can have, with one of four strategies. Inherit keeps the production value. Override sets a preview value from a named source. Derive computes a value from PR metadata. Remove must not exist in a preview.
- Inherit covers values safe to share: port, timeout, pool settings, telemetry settings, the image host address. Override covers the database connection settings, the development Clerk secret key and issuer, and the preview marker. Derive covers the allowed frontend origins. Remove covers the edge secret, the Admin allowlist and the Admin origins, the image bucket write settings, the scan photo bucket settings and the Gemini key.
- The reconcile step lists the PR environment's inherited variables and fails the workflow if any is absent from the manifest. A variable removed from the API's configuration later still needs removing from the manifest, which the check does not enforce.
- Reconciling sets the overrides and derived values in one call with the no-redeploy option. Removals must not trigger a deployment either. If the Railway CLI's delete cannot skip deployments, removals are applied through Railway's API in one batch together with the sets. Setting a removed variable to empty is acceptable only if the API treats an empty value as unset, and that must be confirmed before relying on it.
- The reconcile logic is a script that takes the environment's variable listing and the manifest and produces the sets and removals, so it can be tested without Railway. The workflow calls it.
- The infrastructure definition for Railway stays the production definition. It does not carry preview-specific runtime values; the workflow and manifest own those.

**The preview marker and the boot guard**

- The preview marker is `APP_ENV=preview`, set by the workflow in the same batch as the database settings. It is the single signal for "this environment was provisioned as a preview".
- A new check in configuration loading refuses to continue when Railway reports an environment name other than `production` and the application environment is not `preview`. Both the API and the migration job load the same configuration, so one check protects both. The refusal logs which condition failed and exits.
- Production (its Railway environment is named `production`), local development and CI (no Railway environment name) are unaffected.
- An unprovisioned PR environment inherits `APP_ENV` from production, so it trips the guard. A provisioned one passes. Cancelled, superseded and skipped provisioning runs, which Railway's Wait for CI lets through, can no longer reach the production database.
- The workflow's concurrency changes so provisioning runs are no longer cancelled by later pushes. Runs for one PR queue. A closing event still runs cleanup without being blocked by a stale provisioning run. Railway's own guidance is to keep a workflow that must run before every deploy out of a cancelling concurrency group.

**Cleanup**

- Closing a PR deletes the Neon branch, the preview deployments, the frontend alias and the branch-scoped frontend variables. The Railway PR environment is still deleted by Railway itself.
- Fork and Dependabot PRs are skipped by provisioning and by cleanup, as today.

**PR comment**

- The comment is posted when provisioning finishes, with the Neon branch, the API address and the frontend address, and a line saying the API deploys once checks pass. It is updated in place on later pushes. No separate readiness workflow exists, because it would depend on Railway emitting deployment status events for PR environments, which is unverified.

**Sign-in adoption**

- Identity resolution gains one preview-only branch. When no User has the caller's Auth Identity and the application environment is `preview`, it looks for Users with the same email. If exactly one exists, it rewrites that User's auth provider and auth subject to the caller's Auth Identity and returns that User's profile. If none or several exist, it creates a new User as it does today.
- The lookup and rewrite run inside the existing per-identity advisory lock and transaction, so two concurrent first requests adopt once. Adoption rewrites the copied User in place, so the profile, Collections and Inventory Entries stay attached to the same User row. It does not add a second identity to a User.
- Email comparison is case-insensitive on the stored email. The token's email claim is trusted as verified because the development Clerk instance requires email verification at sign-up (confirmed by the maintainer). No new token claim and no allowlist are added.
- The identity cache is invalidated or bypassed for the adopted User so a stale entry for the old subject cannot resolve to the wrong User.
- No schema change, no new table, and no behaviour change in production. A table mapping several Auth Identities to one User is deliberately deferred until production needs it.
- The glossary's Auth Identity entry still holds. A User row is the Cardstack-side record of one Auth Identity; adoption replaces that identity on the preview's copy, it does not create a second one.

**Documentation**

- The deployment guide's preview section describes the new flow: persistent branch, one deployment, manifest, marker and guard, the reset-by-deleting-the-branch fix and the failed-migration fix.
- The Clerk deployment guide documents adoption, that it is preview-only, and that it depends on the development instance requiring a verified email.
- The preview workflow's header comment stays the authority on what the workflow creates and deletes and is updated to match.

## Testing Decisions

A good test checks behaviour a caller or operator can observe: whether the process boots, which User a caller resolves to, which variable changes a script emits. It does not assert on internal calls, step order inside the workflow, or the text of log lines beyond the one reason a guard refuses.

Three seams are used.

1. **Configuration loading** carries the boot guard. Table-driven cases over the Railway environment name and the application environment: production boots, a PR environment with the preview marker boots, a PR environment without it is refused, and no Railway environment name (local, CI) boots. Prior art is the existing configuration tests, which already cover refusal to boot on bad settings.
2. **User resolution** carries adoption, tested against real Postgres per ADR-0005 in the same style as the existing resolve-caller cases. Cases: an unknown identity with `APP_ENV` not `preview` creates a new User; with `preview` and exactly one email match adopts that User and keeps its profile and Collections; with `preview` and no match creates a new User; with `preview` and several matches creates a new User; two concurrent first requests produce one adopted User; the email match ignores case; a later request with the adopted identity resolves directly. Prior art is the existing resolve-caller tests and the per-identity lock they already exercise.
3. **The reconcile script** carries the manifest logic, driven by fixture variable listings. Cases: every inherited variable is classified and the expected sets and removals are produced; an unclassified inherited variable fails; a remove-strategy variable present is removed and one already absent is a no-op; the derived allowed-origins value is computed from the PR number; running it twice produces the same result. There is no shell test precedent in the repository, so this seam sets the pattern: a plain script run against fixtures in the existing checks, with no new test framework.

The workflow file itself has no automated test. It is checked by `yamllint` per the repository rules and proven by a live PR run, which the first PR's own pull request exercises because workflow changes in a PR take effect for that PR. The live run must confirm: one API deployment per push, the Neon branch kept across two pushes, the removed variables absent, and the guard refusing an environment that skips provisioning.

## Out of Scope

- Custom domains for the API or the frontend, and any DNS automation.
- A workflow that refreshes a preview's data. Deleting the Neon branch is the refresh.
- A readiness workflow driven by Railway deployment events.
- Sanitising or anonymising the production data that a preview branch copies.
- A table that maps several Auth Identities to one User, and any adoption outside preview environments.
- Out-of-order migration support in the migrator.
- An allowlist for development Clerk sign-ups.
- Changes to the production Railway environment, the production frontend, or how the Admin app is deployed.
- Preview environments for the Admin app.

## Further Notes

- Railway's Wait for CI looks at the conclusion of each workflow, not at jobs, and ignores skipped and neutral runs; a cancelled run blocks only when no other workflow on the commit succeeded. The boot guard exists because of this, not because Wait for CI is expected to fail.
- Railway only provisions a domain in a PR environment automatically when the base environment's service uses a Railway-provided domain. This is why the API keeps its Railway domain. Confirm on the first live run that the PR environment's domain exists before the API deploys, as the current workflow already assumes.
- Whether the Railway CLI's variable delete can skip deployments is not confirmed by its documentation. Verify against the pinned CLI version before choosing between the CLI and Railway's API for removals.
- The pinned Railway CLI version in the workflow is newer than the one on the maintainer's machine; check flags against the pinned version, not the local one.
- Bot pull request environments are off by default in Railway. Keep them off; the guard is the safety net if that ever changes.
- The Railway environment named `production` is confirmed by the maintainer.
- Delivery order: the first pull request is verified by its own live run before the second begins.
