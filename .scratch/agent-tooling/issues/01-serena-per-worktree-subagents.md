# 01: Serena for worktree subagents (research)

**Category:** research
**Status:** needs-triage

**Blocked by:** none.

## Agent Brief

**Category:** research
**Summary:** Find out whether a subagent working in its own git worktree can get its own Serena instance rooted at that worktree, so the whole team could use Serena's symbol tools again. Today the fallback is policy: the root agent uses Serena, subagents use the built-in tools (`docs/agents/conventions/serena.md`, "Who uses Serena"). This ticket decides whether that fallback can be retired. Capture findings with the `research` skill under `.scratch/agent-tooling/research/`.

**Why the fallback exists:** Serena starts once per session (`.mcp.json`: `serena start-mcp-server --context claude-code --project-from-cwd`) and stays rooted at the main checkout. `claude-code` is a single-project context, so `activate_project` is disabled. A subagent still sees the shared `mcp__serena__*` tools. Calling `find_symbol` from a worktree on a file that exists only in that worktree returns `[]` with no error, and an edit would land in the main checkout.

**Constraints already decided (grilling session, 2026-10-05):**
- No shared "switch project" state. Backend and frontend agents run in parallel, so one server calling `activate_project` per agent would redirect another agent's edits. Each worktree needs its own Serena process or its own root.
- A cold language-server start per worktree agent (gopls, tsserver, yaml, with a cold `.serena/cache`) is acceptable for edit-heavy delegated tasks.
- Claude Code first. pi (`.pi/`) is out of scope until Claude Code works.
- Keep the narrowed exception wording in `serena.md` until a fix is proven.

**What is known:**
- Subagent frontmatter supports `mcpServers`, using the same schema as `.mcp.json`. Inline servers start when the subagent starts and stop when it finishes (https://code.claude.com/docs/en/sub-agents.md).
- A subagent starts in the main session's working directory, not in a manually created worktree.
- With `isolation: "worktree"`, the worktree is created under `.claude/worktrees/agent-<id>` and Bash runs inside it (confirmed by the probe below).
- Serena's `--project-from-cwd` walks up from the working directory to the nearest `.serena/project.yml` or `.git`, so a worktree resolves to itself when Serena starts inside it. Serena also takes `--project <path>`.

**What is not known (the open questions):**
1. Why the inline `serena-probe` server's tools never appeared for the subagent: the docs say inline servers are "subject to the trust rule for the agent file's folder", and the trust rule is not described. Check whether `/mcp` shows a pending approval for a project-scope inline server, and whether the server failed to start.
2. What working directory an inline MCP server gets inside a subagent, parent's or the worktree's. Not documented.
3. Whether `${VAR}` and `${CLAUDE_PROJECT_DIR}` expansion applies in frontmatter server definitions. Documented only for `.mcp.json`.
4. If the working directory is the parent's, whether a per-task `--project <worktree path>` can be injected (for example a generated agent definition or `--mcp-config` file per worktree).
5. Whether `isolation: "worktree"` can honour the branch naming convention in `docs/agents/orchestration.md`, or whether adopting it means changing that convention.

**Probe already run (2026-10-05).** Agent definition used, saved here because the file was deleted:

```markdown
---
name: serena-probe
description: Throwaway probe. Reports which directory an inline Serena MCP server is rooted at when the subagent runs in a worktree. Delete after use.
model: haiku
mcpServers:
  serena-probe:
    type: stdio
    command: serena
    args:
      - start-mcp-server
      - --context
      - claude-code
      - --project-from-cwd
---

You are a probe. Do exactly these steps and report the raw results.

1. Run `pwd` and `git rev-parse --show-toplevel` with Bash and report both.
2. Write `backend/internal/probe_marker.go` (package `internal`) containing `func ProbeMarkerZq93() {}` using Bash (`cat > ... <<'EOF'`).
3. Load the `serena-probe` MCP server's tools (they are named `mcp__serena-probe__*`; use ToolSearch with `select:mcp__serena-probe__find_symbol,mcp__serena-probe__get_symbols_overview` if they are deferred). If no `serena-probe` tools exist, report that and stop.
4. Call `find_symbol` for `ProbeMarkerZq93` on the `serena-probe` server. Report whether it was found, and the exact error text if not.
5. Report the exact result. Do not commit anything and do not touch any other file.
```

Spawned with `isolation: "worktree"`. Results:
- `pwd` and the git toplevel were both `.../cardstack/.claude/worktrees/agent-<id>`.
- No `mcp__serena-probe__*` tools existed for the subagent, deferred or not. Only the parent's 21 `mcp__serena__*` tools were visible.
- `mcp__serena__find_symbol` (the shared server) for `ProbeMarkerZq93` with `relative_path: backend` returned `[]`, no error, although the file existed in the worktree.

**Candidate approaches to evaluate:**
- Inline `mcpServers` on the component agents, with `--project-from-cwd` (needs open questions 1 to 3 answered).
- A per-worktree `.mcp.json` or `--mcp-config` carrying `--project <worktree path>`, generated by the orchestrator when it creates the worktree.
- A non-single-project Serena context with serialized `activate_project` calls: rejected for parallel agents by the constraint above, listed only to close it off.

**Acceptance criteria:**
- [ ] The open questions above each have an answer with a source (doc URL, or an observed result with the exact command and output).
- [ ] A recommendation: adopt one approach, or confirm that the built-in-tools fallback stays, with the reason.
- [ ] If an approach works, a repeatable check: a worktree subagent writes a fresh marker symbol and finds it through its own Serena instance, and a symbol edit through it changes the worktree and leaves the main checkout untouched.
- [ ] If an approach is adopted, the follow-up changes to `serena.md`, `orchestration.md` and the two agent files are listed as a separate ticket; this ticket changes no code or docs besides its research note.

**Out of scope:**
- pi support.
- Changing the Serena context for the root agent.
- Excluding Serena from subagents mechanically. The docs confirm `disallowedTools: mcp__serena` (server-level) in subagent frontmatter; whether it is adopted is a separate decision.

## Comments

Grilling session, 2026-10-05 — decided to fall back to plain tools for subagents and park the per-worktree question as this ticket. The fallback is in `docs/agents/conventions/serena.md` ("Who uses Serena"), the two agent files and `AGENTS.md`.
