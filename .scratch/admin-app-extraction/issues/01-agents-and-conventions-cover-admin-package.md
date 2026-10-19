# 01: Agents and conventions cover the admin package

**What to build:** The repository says a second frontend package, `admin/`, exists and that the frontend rules apply to it, so an agent can be given work there. The `frontend-agent` definition and the orchestration doc list `admin/` in its scope (today it is restricted to `frontend/`), the frontend conventions state that they hold for both packages, and the Serena and security docs carry over whatever is path-specific. Nothing else changes: no code, no `admin/` directory yet.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] The `frontend-agent` definition and `docs/agents/orchestration.md` name `admin/` as part of the frontend scope, with the same restrictions as `frontend/`
- [x] `docs/agents/conventions/frontend.md` says it applies to `frontend/` and `admin/`
- [x] Any path-specific rule in the security and Serena docs (permissions, hooks, allowed paths) that names `frontend/` also covers `admin/`, or the ticket records why it does not
- [x] `yamllint` passes on any YAML touched, and no markdown line is hand-wrapped

## Answer

`.claude/agents/frontend-agent.md` (and its `.pi/agents/` symlink), `docs/agents/orchestration.md` (big-task definition, launch-message scope, scope check) and `docs/agents/conventions/frontend.md` now name `admin/` alongside `frontend/`. The security doc and `.claude/settings.json` hold no path-specific rule that names `frontend/` (the hooks and permissions are repo-wide), and `docs/agents/conventions/serena.md` names none either; Serena's `typescript` language server and `scripts/lsp/frontend.sh` already cover any TypeScript package, so nothing changes there. Not done, because `admin/` does not exist yet: `scripts/bootstrap-worktree/frontend.sh` and `scripts/verification/frontend.sh` only run in `frontend/`, so the ticket that creates `admin/` (03) must extend them, and the agent's read-only script exception already names them.
