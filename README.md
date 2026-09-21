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
