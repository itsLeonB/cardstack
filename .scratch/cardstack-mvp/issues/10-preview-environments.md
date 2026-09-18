# 10: Per-PR preview environments

**What to build:** Wire the same Neon/Railway/Vercel per-PR preview pipeline cashus already runs (`.github/workflows/preview-environments.yml` + `scripts/setup-preview-environments.sh`) into cardstack: a Neon DB branch forked per PR, credentials pushed into that PR's Railway environment (single `cardstack` service — MVP has no worker/job process), and a Vercel preview build pointed at it. Requires a one-time manual setup (Neon/Railway/Vercel projects created, "PR Deploys" enabled, secrets set) via the ported wizard script before the workflow can run for real.

**Blocked by:** 01 (needs the backend Dockerfile/build and frontend build the workflow deploys)

**Status:** ready-for-agent

- [ ] `.github/workflows/preview-environments.yml` ported from cashus, adjusted to cardstack's single Railway service and repo/service names
- [ ] `scripts/setup-preview-environments.sh` (one-time wizard) ported and adjusted the same way
- [ ] `docs/deployment/preview-environments.md` documents what gets provisioned and the one-time setup steps
- [ ] One-time setup actually run (Neon project, Railway project with PR Deploys enabled, Vercel project, secrets set) — human step, not agent-automatable
- [ ] A test PR confirms the full provision → comment → cleanup cycle works end to end
