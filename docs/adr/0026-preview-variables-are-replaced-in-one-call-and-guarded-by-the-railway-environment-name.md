# Preview variables are replaced in one call and guarded by the Railway environment name

Railway copies production's variables, database included, into every PR environment. The preview workflow turns that copy into a preview with a single `variableCollectionUpsert` per push, with `replace: true` and `skipDeploys: true`: the payload is the environment's variable listing without `RAILWAY_*`, plus the overrides and derived values of `backend/infra/.railway/preview-vars.yml`, minus its removals, so the API deploys once, already configured. The two secrets the payload needs (the Neon password and the Clerk development secret key) are filled in by a step that runs no repository code. The API refuses to boot (`config.Load`) when `RAILWAY_ENVIRONMENT_NAME` is set, is not `production`, and `APP_ENV` is not `preview`, so a PR environment the workflow never configured (cancelled, superseded or skipped run) cannot run on production's database; Railway sets the name per environment, so a copied variable cannot carry production's value into a PR environment. A preview runs in gin's release mode, as production does.

## Considered Options

- `railway variable delete` for each removal. Rejected: Railway's `variableDelete` takes no `skipDeploys`, so every removal triggers a deployment of a half-configured environment.
- `railway variable edit --skip-deploys`. Rejected: its apply is one `variableCollectionUpsert` with `skipDeploys` for the sets and then one `variableDelete` per removal without it, so the removals still deploy.
- Setting a removed variable to the empty string. Rejected: the variable would still exist, and nothing confirms the API treats empty as unset.

## Consequences

A sealed variable lists with a null value and cannot be read back, so `replace` would lose it: the run fails unless the manifest removes it or overrides it. `replace` writes back what the listing returned; whether `${{...}}` references come back as references and not as rendered text, and that a push produces exactly one deployment, are checked on a live run (ticket 05). The production Railway environment must stay named `production`.
