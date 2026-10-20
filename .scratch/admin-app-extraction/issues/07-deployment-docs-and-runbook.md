# 07: Deployment docs and runbook

**What to build:** The docs describe the final state and give the operator a checklist for the cutover. The deployment doc gets the new Vercel project and subdomain in the topology, the Admin app's `VITE_*` variables, and the cutover order (set `ADMIN_CLIENT_URLS` in Railway, deploy the API, deploy the Admin app, deploy the user app without `/admin`, with the window where no Admin can write and the rollback by revert). The Clerk doc gets `ADMIN_CLIENT_URLS`, the admin-origin rule and its boot refusals, and updates the section on `/admin` being a page in the user app. The images doc gets the R2 CORS entry for the Admin app's origin. The README lists the package if it lists packages. The domain stays out of the repository.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** 06

**Status:** resolved

- [ ] `docs/agents/deployment.md` routes to the new setting and gives the cutover order, the unreachable window and rollback
- [ ] `docs/agents/deployment/clerk.md` documents `ADMIN_CLIENT_URLS`, what a 404 on an admin route now means for a user-origin token, and the session sharing across subdomains with the satellite-domain caveat
- [ ] `docs/agents/deployment/images.md` lists the Admin app's origin among the origins R2 CORS must allow for PUT
- [ ] The local setup for the Admin app (port, Clerk dev instance, origin in `ADMIN_CLIENT_URLS`) is written down in the environment doc
- [ ] No markdown line is hand-wrapped and `yamllint` passes on any YAML touched

## Comments

- Carry-over from ticket 02: `docs/agents/deployment/clerk.md` still says "Unset, the admin routes do not exist" and does not mention `ADMIN_CLIENT_URLS`; anyone with `ADMIN_EMAILS` set in a local `backend/.env` must now also set `ADMIN_CLIENT_URLS` (example: `http://localhost:3001`) or the API refuses to boot.
