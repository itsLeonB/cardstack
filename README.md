# cardstack

## Environment Setup

### Linters

```sh
pip install --user yamllint
```

### GitHub Repository Secrets

These are the required repository secrets needed for [the preview environment provisioning](./.github/workflows/preview-environments.yml) to work.

| Secret | Where to get it |
| --- | --- |
| `NEON_API_KEY` | [console.neon.tech/app/settings?modal=create_api_key](https://console.neon.tech/app/settings?modal=create_api_key) |
| `NEON_DATABASE_NAME` | Neon dashboard |
| `NEON_PROJECT_ID` | `https://console.neon.tech/app/projects/<project-id>` |
| `NEON_ROLE_NAME` | Neon dashboard |
| `RAILWAY_API_TOKEN` | [railway.com/account/tokens](https://railway.com/account/tokens) |
| `RAILWAY_ENVIRONMENT_ID` | `https://railway.com/project/<project-id>?environmentId=<environment-id>` |
| `RAILWAY_PROJECT_ID` | `https://railway.com/project/<project-id>` |
| `VERCEL_ORG_ID` | `https://vercel.com/<team-slug>/~/settings` |
| `VERCEL_PROJECT_ID` | `https://vercel.com/<team-slug>/<project-name>/settings` |
| `VERCEL_TOKEN` | [vercel.com/account/settings/tokens](https://vercel.com/account/settings/tokens) |

### Clerk (sign-in provider)

Run `scripts/clerk-setup.sh` to configure the Clerk development instance and store its keys locally and in GitHub. It is re-runnable and skips finished stages. Once the domain from ticket 02 exists, run `scripts/clerk-setup.sh production` for the live instance, Railway and Vercel.

### Cloudflare, R2 and hosting

Run `scripts/cloudflare-setup.sh` to put the domain on Cloudflare, create the R2 bucket and its `cdn` subdomain, proxy the `api` subdomain, add the edge secret header and per-IP rate rule, and walk you through setting the resulting variables on Railway and Vercel (secrets go on your clipboard, never on a command line). It is re-runnable and skips finished stages; its values are kept in a private file under `~/.local/state/cardstack/`, outside the repo.
