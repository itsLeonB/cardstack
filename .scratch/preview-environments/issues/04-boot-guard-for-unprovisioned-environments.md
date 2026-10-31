# 04: Boot guard for unprovisioned PR environments

**What to build:** A PR environment that was never provisioned fails closed. When Railway reports an environment name other than `production` and the application environment is not `preview`, configuration loading refuses to continue, so neither the API nor the pre-deploy migration can touch the production database that an unprovisioned environment inherits. This covers runs that Railway's Wait for CI lets through: cancelled, superseded or skipped provisioning. Production, local development and CI are unaffected. Part of the first pull request.

**Blocked by:** 03 (the workflow must set `APP_ENV=preview` first, or every preview would refuse to boot).

**Status:** done

- [ ] With the Railway environment name `production`, configuration loads regardless of the application environment.
- [ ] With a non-production Railway environment name and `APP_ENV=preview`, configuration loads.
- [ ] With a non-production Railway environment name and any other `APP_ENV`, configuration loading fails, and the error says which condition tripped it.
- [ ] With no Railway environment name (local development, CI), configuration loads.
- [ ] The API and the migration job both load configuration through the guarded path, so the pre-deploy migration is protected too.
- [ ] The guard is covered by a table-driven test next to the existing configuration tests (seam 1).
- [ ] The deployment guide documents the guard and what a refusal to boot in a PR environment means.
