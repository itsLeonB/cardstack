# Hooks, permissions and security

`.claude/settings.json` enforces the hooks and permissions below. Put personal overrides in `.claude/settings.local.json`, which isn't shared.

## Enforced settings

- **Serena hooks**: `serena-hooks remind` nudges an agent that reads code with built-in tools; `activate`, `auto-approve` and `cleanup` manage the Serena session. See `docs/agents/conventions/serena.md`.
- **`.claude/hooks/block-commit-on-main.sh`**: refuses `git commit` and `git merge` on a protected branch, and `git push` from or to one. A command that creates a branch first is allowed. The `PROTECTED` variable at the top of the script lists the branches (main).
- **`deny-big-output`**: refuses `cat` or `git diff` output over 20 KB, and a `Read` over 40 KB without `offset` or `limit`. The message names the cheaper route (`--stat`, `offset`/`limit`, `ctx_execute_file`). Prefix a Bash command with `BIG_OK=1 ` to override. Source: `tools/cmd/deny-big-output`; install: `tools/README.md`. Without the binary the hook errors but doesn't block.
- **`permissions.deny`**: patterns the harness refuses. A wider allow in `settings.local.json` doesn't override a deny.
- **`CLAUDE_CODE_DISABLE_ADVISOR_TOOL`**: turns off the native advisor tool, so the advisor subagent is the only advisor.
- **hookify** (`hookify@claude-plugins-official`): turns a mistake into a rule file.
- **security-guidance** (`security-guidance@claude-plugins-official`): reviews code in three layers, configured under `env` in `.claude/settings.json`. The plugin sends diffs and the files it reads to your model endpoint. The plugin's README lists every variable.

## security-guidance layers

| Layer | Variable | State |
|---|---|---|
| Pattern warnings on `Edit` and `Write` | `ENABLE_PATTERN_RULES` | on |
| LLM review of the diff at turn end | `ENABLE_STOP_REVIEW`, `SECURITY_REVIEW_MODEL` | off |
| Agentic review at commit | `ENABLE_COMMIT_REVIEW`, `SG_AGENTIC_MODEL` | on, `claude-sonnet-5-5` |
| Doubled reviews for higher recall | `SG_DUAL_OR` | off |

Both reviews cost tokens: they use the newest Opus unless pinned. Project-specific rules go in `.claude/claude-security-guidance.md`, which the turn-end review reads.

## Add a hookify rule

1. Run `/hookify` right after an agent repeats a mistake. With no argument it reads the conversation and proposes rules.
2. Check the rule file at `.claude/hookify.<name>.local.md`: markdown with YAML frontmatter that matches a tool call by event and regex, then warns or blocks.

A new rule applies on the next tool use, with no restart. `/hookify:list` shows rules; `/hookify:configure` toggles them.

Target mistakes that tooling can't catch and that have a precise textual signature (a command, a path, a string in an edit). Don't duplicate Serena's `remind` (built-in code reads) or `block-commit-on-main` (commits on protected branches).

## Secrets

Real values live in `.envrc` (gitignored, see `docs/environment.md`). Never put them in `.mcp.json`, settings files or docs; reference them as `${VAR}`.
