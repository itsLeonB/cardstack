# 01: Serena for worktree subagents (research)

**Category:** enhancement
**Status:** done — fallback stays; findings in `.scratch/agent-tooling/research/serena-per-worktree-subagents.md`. Per-worktree Serena only works for a separate headless `claude -p` process, not for Agent-tool subagents.

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

> *This was generated by AI during triage.*

Triage, 2026-10-06: moved `needs-triage` to `ready-for-human`, category `enhancement`. The agent brief above stands as written. No existing per-worktree Serena setup was found (`.mcp.json` has one shared `serena` server), and `.out-of-scope/` has no prior rejection.

**Why this can't be delegated to an AFK agent:**
- Open question 1 needs a human to run `/mcp` and look for a pending approval on a project-scope inline server. An agent can't see that interactive UI.
- A new probe agent definition may need a session restart or `/agents` reload before it can be spawned.
- Acceptance criteria 1 to 3 need a live Claude Code session with the maintainer supervising.

The 2026-10-05 probe result (shared server returns `[]` for a worktree-only file) was not re-run during triage.

Maintainer observation, 2026-10-06: opening `/mcp` showed no pending approval. Caveat: the `serena-probe` agent file was already deleted, so this does not rule out a trust prompt for a live project-scope inline server. Open question 1 stays open until the probe is recreated and `/mcp` is checked while it runs.

> *This was generated by AI during triage.*

Probe re-run, 2026-10-06 (answers open question 1 in part): recreated `.claude/agents/serena-probe.md` exactly as saved above, restarted Claude Code so the agent type loaded, then spawned it with `subagent_type: "serena-probe"` and `isolation: "worktree"`.
- `pwd` and `git rev-parse --show-toplevel`: both `.../cardstack/.claude/worktrees/agent-a360fcd4aebfe3cc6`.
- `backend/internal/probe_marker.go` was written in the worktree.
- ToolSearch `select:mcp__serena-probe__find_symbol,mcp__serena-probe__get_symbols_overview` returned nothing. No `serena-probe` tools exist for the subagent, so the 2026-10-05 result reproduces.
- `/mcp` was opened twice while the probe ran (it finished in about 25 s): no pending approval and no `serena-probe` entry.
- `.claude/settings.local.json` has `enableAllProjectMcpServers: true`, so a name allowlist is not the cause.
- `ls -d ~/.cache/claude-cli-nodejs/-home-leon-Projects-itsLeonB-cardstack/mcp-logs-*probe*` found nothing, so no start attempt was logged for a `serena-probe` server. The server most likely never started, with no visible prompt or error. Not proven: the log naming for inline servers is unconfirmed.

Not yet tried: `${CLAUDE_PROJECT_DIR}` expansion (question 3), the working directory of an inline server (question 2), and a per-worktree `--mcp-config` (question 4).

> *This was generated by AI during triage.*

Candidate 2 test, 2026-10-06: **works, but only as a separate `claude -p` process, not as an Agent-tool subagent.** `--mcp-config` is a CLI flag, and Agent-tool children inherit the parent's MCP servers, so it cannot reach them.

Config (`serena-wt.json`, in the session scratchpad):

```json
{"mcpServers":{"serena-wt":{"type":"stdio","command":"serena","args":["start-mcp-server","--context","claude-code","--project","/home/leon/Projects/itsLeonB/cardstack/.claude/worktrees/agent-a360fcd4aebfe3cc6"]}}}
```

Command (run from the probe worktree, in manual permission mode):

```
claude -p --model haiku --mcp-config serena-wt.json --strict-mcp-config --allowedTools 'mcp__serena-wt' --permission-mode dontAsk --output-format json "Use only the serena-wt MCP tools. 1) find_symbol ProbeMarkerZq93 (relative_path backend). 2) replace_symbol_body so the body is: func ProbeMarkerZq93() { _ = 1 } ..."
```

Results (4 turns, `is_error: false`):
- `find_symbol` returned `[{"name_path":"ProbeMarkerZq93","kind":"Function","relative_path":"backend/internal/probe_marker.go","body_location":{"start_line":2,"end_line":2},"body":"func ProbeMarkerZq93() {}"}]`. The worktree-only symbol was found, so the instance is rooted at the worktree.
- `replace_symbol_body` returned `OK`.
- Checked outside the child: the worktree's `backend/internal/probe_marker.go` now reads `func ProbeMarkerZq93() { _ = 1 }`. `ls` of the same path in the main checkout: no such file. `git status --short` in main lists only this ticket and `.claude/agents/serena-probe.md`.

This meets acceptance criterion 3 for the headless-process route. Adopting it for `backend-agent` and `frontend-agent` would mean launching them through Bash instead of the Agent tool, which changes `orchestration.md` and gives up background notifications, SendMessage and the agent-file tool restrictions. Open decision for the maintainer.

Not tested: the worktree's own tracked `.mcp.json` with `--project-from-cwd` (would avoid generating a config per worktree), open questions 2 and 3.

Note: the first two attempts at this run were blocked by the auto-mode classifier ("Create Unsafe Agents"), including after the maintainer's verbal go-ahead. The run only went through after switching to manual permission mode.

> *This was generated by AI during triage.*

Tracked `.mcp.json` test, 2026-10-06: **the read half works; the edit half was blocked by a project hook, not by Serena.**

Command (from the probe worktree; `.mcp.json` is tracked and identical to the main checkout's, so the server is `serena` with `--project-from-cwd`):

```
claude -p --model haiku --mcp-config .mcp.json --strict-mcp-config --allowedTools 'mcp__serena' --permission-mode dontAsk --output-format json "Use only the serena MCP tools. 1) find_symbol ProbeMarkerZq93 (relative_path backend). 2) replace_symbol_body so the body is: func ProbeMarkerZq93() { _ = 2 } ..."
```

Results (4 turns, `is_error: false`):
- `find_symbol` found `ProbeMarkerZq93` at `backend/internal/probe_marker.go:2` with body `func ProbeMarkerZq93() { _ = 1 }`. That body exists only in the worktree, so `--project-from-cwd` resolved to the worktree when the process started there.
- `replace_symbol_body` was refused by the `require-advisor.py` PreToolUse hook ("no advisor call is saved in transcript"). The hook is loaded from the worktree's tracked `.claude/settings.json`. The edit did not happen: the worktree file still reads `_ = 1`, and the main checkout still has no `probe_marker.go`.

The edit through a `--project-from-cwd` server was not exercised. The generated-config run above did exercise `replace_symbol_body` against the same kind of server, so the remaining risk is low, but it is untested for this config. Repeating it needs the hook satisfied inside the child, which I did not attempt.

Consequence for the recommendation: for a headless child started in a worktree, the tracked `.mcp.json` is enough for reads and the orchestrator does not need to generate a config per worktree. Unlike `--mcp-config` with `--project`, this depends on the child's working directory being the worktree.

## Answer

> *This was generated by AI during triage.*

Keep the built-in-tools fallback for subagents; revisit if inline `mcpServers` in agent frontmatter start working. Per-question answers, sources and the recommendation are in `.scratch/agent-tooling/research/serena-per-worktree-subagents.md`.

- Inline frontmatter servers never started (no tools, no prompt, no log), so questions 2 and 3 are moot and question 1's cause is unknown.
- Question 4: a per-worktree `--project` config works, but only for a separate `claude -p` process. Adopting that for the component agents is the maintainer's call, because it changes `orchestration.md` and loses notifications, SendMessage and the tool restrictions, and the child hits `require-advisor.py`.
- Question 5: `isolation: "worktree"` branches are named `worktree-agent-<id>`, which does not match `<semantic branch>/<branch name>` in `orchestration.md`.
- Acceptance criterion 4 does not apply, since no approach is adopted. One follow-up remains: `docs/agents/conventions/serena.md` (line 10) still says per-worktree Serena is "open research" and should link to the note instead.
- The probe worktree, its branch and `.claude/agents/serena-probe.md` were deleted after the run.
