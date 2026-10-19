# 01: Agents and conventions cover the admin package

**What to build:** The repository says a second frontend package, `admin/`, exists and that the frontend rules apply to it, so an agent can be given work there. The `frontend-agent` definition and the orchestration doc list `admin/` in its scope (today it is restricted to `frontend/`), the frontend conventions state that they hold for both packages, and the Serena and security docs carry over whatever is path-specific. Nothing else changes: no code, no `admin/` directory yet.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The `frontend-agent` definition and `docs/agents/orchestration.md` name `admin/` as part of the frontend scope, with the same restrictions as `frontend/`
- [ ] `docs/agents/conventions/frontend.md` says it applies to `frontend/` and `admin/`
- [ ] Any path-specific rule in the security and Serena docs (permissions, hooks, allowed paths) that names `frontend/` also covers `admin/`, or the ticket records why it does not
- [ ] `yamllint` passes on any YAML touched, and no markdown line is hand-wrapped
