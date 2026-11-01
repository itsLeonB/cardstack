# Claude-only configs in this repo, and where each one goes

TL;DR: nine Claude-only things exist; five have a clean home already (skills are generic via `.agents/skills`, `.mcp.json` is read by Pi and Cursor as well as Claude, role files are just Markdown pointers), three need a Pi-native rewrite (hooks → extension events, `permissions.deny` → a `tool_call` extension, `/implement` → a Pi prompt template), and two have no portable equivalent at all (hookify, security-guidance — Claude plugin marketplace only). Nothing here needs a new abstraction: the generic paths already exist in the repo.

Checked: Pi `1.4.0` docs and bundled examples at `~/.pi/agent/install/releases/1.4.0/node_modules/@earendil-works/pi-coding-agent/`; every claim below cites the file it came from. Web sources are listed in section 6.

## 1. Inventory: what exists only for Claude

| Path | What it is | Claude-only mechanism |
|---|---|---|
| `.claude/agents/advisor.md`, `backend-agent.md`, `frontend-agent.md` | Role definitions. Frontmatter `name`, `description`, `model`, `color`, then the body | Claude Code custom subagents |
| `.claude/skills/` (37 symlinks → `../../.agents/skills/<name>`) | Discovery path so Claude Code finds the shared skills | Claude Code's skill directory |
| `.claude/settings.json` | `env` (`CLAUDE_CODE_DISABLE_ADVISOR_TOOL`, `ENABLE_STOP_REVIEW`, `SG_AGENTIC_MODEL`), `hooks` (`PreToolUse`, `SessionStart`, `SessionEnd` running `serena-hooks … --client=claude-code`), `enabledPlugins` (`hookify`, `security-guidance`), `permissions.deny` | Claude Code settings + plugin marketplace |
| `.claude/settings.local.json` | `permissions.allow`, `enableAllProjectMcpServers`, `enabledMcpjsonServers` | Claude Code per-user settings |
| `.claude/.gitignore` | Ignores `worktrees/` | Claude Code `--worktree` directory |
| `.claude/hookify.<name>.local.md`, `.claude/claude-security-guidance.md` | Rule files the two plugins read (referenced by `docs/security.md`, not present in the repo) | Claude Code plugins |
| `AGENTS.md:25`, `docs/agents/workflows/implement.md:38,42,76`, `implement-spec.md:21`, `.claude/agents/*-agent.md:12,27,45,47` | Pointers to `.claude/agents/*.md` and `.claude/skills/` | Path convention, not a feature |
| `docs/security.md` (whole file), `README.md:8` | Documentation organised around `.claude/settings.json`, hookify, security-guidance | Docs |
| `scripts/setup-environment.sh:53` and comments at `:3,5,16` | `~/.claude/.i-have-adhd-always`, `-a claude-code` in every `npx skills add` | Claude Code |
| `frontend/.oxlintrc.json:25`, `admin/.oxlintrc.json:25` | `.claude/**` in the ignore list, next to `.codex/**`, `.cursor/**`, `.gemini/**`, `.pi/**` | Harmless; no action needed |

Already stale, independent of this migration: `backend-agent.md:27` and `frontend-agent.md:27` tell the agent to load skills from `.claude/skills/`, which are symlinks to `.agents/skills/`. The real path is `.agents/skills/`.

## 2. Agent role files (`.claude/agents/*.md`)

Generic alternative: none. There is no cross-vendor format for a role file with its own model and tool allowlist — `AGENTS.md` carries project instructions, and the Agent Skills spec carries workflows, but neither carries agent definitions (`https://agents.md`; `https://agentskills.io/specification`).

The neighbours, for comparison:

- Codex: standalone TOML files under `~/.codex/agents/` or `.codex/agents/`, e.g. `.codex/agents/reviewer.toml` (`https://developers.openai.com/codex/subagents`, "To define your own custom agents, add standalone TOML files…").
- Pi: the bundled `subagent` example reads Markdown with frontmatter, `name` / `description` / `tools` / `model`, from `~/.pi/agent/agents/` and — with `agentScope: "both"` — `<cwd>/.pi/agents/` (`examples/extensions/subagent/agents.ts:119,129`; `index.ts:476-477`). Closest to the Claude format, but there is no `color` field and `model` must be a Pi model id.

Pi-native: `.pi/agents/<name>.md`, same frontmatter minus `color`, delivered by installing the `subagent` example as an extension. That extension is not installed here — `.pi/settings.json` holds only `npm:@lystran/pi-serena-hooks`, and `.pi/extensions/` holds the two guard extensions.

Recommendation: this repo's workflow never auto-discovers the role files. The orchestrator reads one and pastes it into the worker's task message (`docs/agents/workflows/implement.md:76`), so no harness needs to find them by path. Move them to `docs/agents/roles/` beside the other agent docs, drop `color:`, and update the six pointer sites. That keeps one copy and no symlinks. Add `.pi/agents/` symlinks only if the Pi subagent extension is installed later.

## 3. Skills (`.claude/skills` symlinks)

Generic alternative: already in use. Pi reads `.agents/skills/` directly, project-scoped and ancestor-scoped (`docs/skills.md`, "Pi also supports the Agent Skills locations `~/.agents/skills/` and `.agents/skills/`"). Codex documents the same path in its own subagent example (`path = "~/.agents/skills/docs-editor/SKILL.md"`, `https://developers.openai.com/codex/subagents`). The format is the published spec (`https://agentskills.io/specification`).

Action: the 37 symlinks exist only so Claude Code can find them. If Claude Code stays in use, keep them; otherwise `git rm` the directory. `.agents/skills/` itself does not change. Vendored third-party skills that hardcode `${CLAUDE_PLUGIN_ROOT}` (`ui-ux-pro-max`, 11 lines) come from upstream and are not worth editing; they affect only whoever runs that skill in a harness that does not export the variable.

## 4. Hooks (`.claude/settings.json` → `hooks`)

Generic alternative: none. Every harness has its own file and its own event names:

- Claude Code: `settings.json` `hooks` with `PreToolUse`, `SessionStart`, `SessionEnd` (current `.claude/settings.json`).
- Codex: `hooks.json` or `[hooks]` in `config.toml`, discovered beside the active config layer; events `PreToolUse`, `PostToolUse`, `SessionStart`, `SessionEnd`, `UserPromptSubmit`, `SubagentStart`, `Stop` (`https://learn.chatgpt.com/docs/hooks.md`).
- Gemini CLI: `settings.json` `hooks`, events `BeforeTool`, `BeforeModel`, … with exit-code-2 blocking and `BeforeToolSelection` (`https://github.com/google-gemini/gemini-cli/blob/main/docs/hooks/reference.md`).
- Cursor: `.cursor/hooks.json`, stdio JSON both ways (`https://cursor.com/docs/hooks`).

Pi-native: extension events. `tool_call` authorises or blocks a call, `tool_result` rewrites a result, `session_start` / `session_shutdown` manage session scope, `user_bash` handles shell commands (`docs/extensions.md:159-163`, `:240`). Handler failure blocks the tool as a fail-safe (`docs/extensions.md:375`). The bundled `permission-gate.ts` example is exactly the "authorise or block" pattern (`docs/extensions.md:390`).

What this repo already has: `@lystran/pi-serena-hooks` bridges Serena's hook commands to Pi's tool events, including `remind` before `grep` and `multi_grep` (`docs/serena.md:54-58`). So the Claude hook block has no Pi counterpart to write — it is already replaced. No `--client=pi` equivalent is needed.

## 5. `permissions.deny` and `permissions.allow`

Generic alternative: none. Pi has no permission file and does not ask for approval before a tool call: "it does not ask for approval before every tool call" (`docs/security.md`, "Run Pi safely"). The documented way to refuse a call is an extension that blocks the `tool_call` event, not a settings list.

Pi-native: `.pi/extensions/block-commit-on-main.ts` and `.pi/extensions/deny-big-output.ts` already do this job, and `docs/security.md:8-9` records them as Pi-only with no Claude counterpart — the direction of that sentence is backwards today. The Claude deny list is empty (`.claude/settings.json` `"deny": []`), so nothing is lost by deleting it. The allow list in `settings.local.json` (`Bash(python3 *)`, `mcp__postgres-prod`, `Bash(rtk grep *)`) had no effect on Pi anyway.

## 6. MCP servers (`.mcp.json`)

Generic alternative: this is it. Pi reads a project `.mcp.json` and gates it on project trust (`docs/mcp.md:32` and the conversion table at `docs/mcp.md:110`, which names Claude Code, Cursor and Claude Desktop as sources of the same `mcpServers` shape). Keep the file, keep the `${VAR}` references.

Pi-native difference: `enableAllProjectMcpServers` and `enabledMcpjsonServers` have no Pi counterpart — trust is per-directory, saved in `~/.pi/agent/trust.json`, set with `/trust` (`docs/security.md`, "How Pi chooses a trust decision"). Delete both keys with `settings.local.json`.

## 7. Slash commands (`/implement`, `/implement-spec`)

Generic alternative: Agent Skills. `docs/agents/workflows.md` calls them "the maintainer's slash commands", and the repo already ships the two skills at `.agents/skills/implement/` and `.agents/skills/implement-spec/`.

Pi-native: a skill is invoked as `/skill:implement` (`docs/skills.md`, "Arguments after `/skill:name`…"). To keep the exact `/implement` spelling, add `.pi/prompts/implement.md` and `.pi/prompts/implement-spec.md` as prompt templates — a template's filename is its command name and it can forward `$ARGUMENTS` (`docs/prompt-templates.md`). That is two small files; skip them if `/skill:implement` is acceptable.

## 8. Plugins (hookify, security-guidance)

Generic alternative: none. Both are Claude Code marketplace plugins (`hookify@claude-plugins-official`, `security-guidance@claude-plugins-official` in `.claude/settings.json`). Nothing in the Pi, Codex, Gemini or Cursor docs offers a drop-in.

Pi-native, ranked by cost:

1. Delete. The commit-review layer is the only part with real value, and CI already runs the checks.
2. Port only what earns its keep. A `turn_end` or `agent_before_settle` extension can review the diff and request one continuation (`docs/extensions.md:230`); that replaces `ENABLE_COMMIT_REVIEW`. `block-commit-on-main.ts` is a working example of the same shape.
3. Port hookify: a `session_start` extension that lists `.agents/rules/*.md` in the system prompt. The bundled `claude-rules.ts` example already does precisely that for `.claude/rules/` (`examples/extensions/claude-rules.ts:1-22`).

`docs/security.md:11-13,19-31` and `README.md:8` are the only places documenting these; both get rewritten either way.

## 9. Environment variables

`CLAUDE_CODE_DISABLE_ADVISOR_TOOL` exists to keep the advisor subagent the only advisor. Its Pi analogue does not exist: Pi has no native advisor tool. `ENABLE_STOP_REVIEW` and `SG_AGENTIC_MODEL` belong to security-guidance. All three go with section 8.

## 10. Recommendation, smallest complete change

1. `git mv .claude/agents/*.md docs/agents/roles/`, strip `color:` from the three frontmatters.
2. Update the six pointers: `AGENTS.md:25`, `docs/agents/workflows/implement.md:38,42,76`, `implement-spec.md:21`, and the `Scope` / `Skills` lines inside the two component role files.
3. Fix `.claude/skills/` → `.agents/skills/` in the two component role files while there.
4. Rewrite `docs/security.md` to Pi only: the extensions, project trust, `.mcp.json`, the Serena hook package. Delete the hookify and security-guidance sections. Update `README.md:8`.
5. `git rm -r .claude/`, `git rm .claude/settings.json .claude/settings.local.json` (part of the same step).
6. `scripts/setup-environment.sh`: drop the `~/.claude` line and the three Claude-flavoured comments; keep `-a claude-code -a codex` in the `npx skills add` lines only if Claude Code stays.
7. Optional: `.pi/prompts/implement.md`, `implement-spec.md` for the exact `/implement` spelling.

Steps 1–6 touch 12 files and no code. About 30 minutes; the rewrite of `docs/security.md` is most of it.

Open decision that changes steps 5 and 6: whether Claude Code is still in use by the maintainer or in CI. If it is, keep `.claude/skills/` and `-a claude-code`, and move the role files instead of deleting `.claude/agents/`.

## 11. Sources

- `https://agents.md` — AGENTS.md scope, adoption, nesting rules.
- `https://agentskills.io/specification` — Agent Skills directory and frontmatter spec.
- `https://developers.openai.com/codex/subagents` — Codex custom agents as `.codex/agents/*.toml`, plus the `~/.agents/skills/` path.
- `https://learn.chatgpt.com/docs/hooks.md` — Codex hook files and event names.
- `https://github.com/google-gemini/gemini-cli/blob/main/docs/hooks/reference.md` — Gemini CLI hook events and exit-code contract.
- `https://cursor.com/docs/hooks`, `https://cursor.com/docs/rules` — Cursor hooks file and `.cursor/rules`.
- `https://code.claude.com/docs/en/sub-agents` — Claude Code subagent frontmatter and discovery.
- Pi docs read locally: `docs/skills.md`, `docs/prompt-templates.md`, `docs/slash-commands.md`, `docs/extensions.md`, `docs/mcp.md`, `docs/security.md`, `docs/settings.md`; examples `examples/extensions/subagent/{README.md,agents.ts,index.ts}`, `examples/extensions/claude-rules.ts`.
