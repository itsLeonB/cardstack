# Hooks, permissions and security

What `.claude/settings.json` enforces, and how to add to it. Personal overrides go in `.claude/settings.local.json`, which is not shared.

## What is wired

- **Serena hooks.** `serena-hooks remind` nudges a thread that reads code with built-in tools instead of Serena; `activate`, `auto-approve` and `cleanup` manage the Serena session. See `docs/agents/conventions/serena.md`.
- **`.claude/hooks/block-commit-on-main.sh`.** Refuses `git commit` and `git merge` while the current branch is protected (main), and `git push` from or to a protected branch. A command that first creates a branch is allowed. The protected list is the `PROTECTED` variable at the top of the script.
- **`permissions.deny`.** Patterns the harness refuses outright. Edit the list in `.claude/settings.json`; a wider allow in `settings.local.json` does not override a deny.
- **`CLAUDE_CODE_DISABLE_ADVISOR_TOOL`.** Turns off the native advisor tool, so the advisor T3 thread is the only advisor.
- **hookify.** `hookify@claude-plugins-official` turns a mistake into a rule file.
- **security-guidance.** `security-guidance@claude-plugins-official` reviews code in three layers: instant pattern warnings on `Edit` and `Write` (`ENABLE_PATTERN_RULES`), an LLM review of the diff when a turn ends (`ENABLE_STOP_REVIEW`, model `SECURITY_REVIEW_MODEL`), and an agentic review at commit time (`ENABLE_COMMIT_REVIEW`, model `SG_AGENTIC_MODEL`). The two reviews call a model, the newest Opus unless pinned, so they cost tokens. `SG_DUAL_OR=on` doubles each review for higher recall. Pattern warnings on, turn-end review off, commit review on and pinned to `claude-sonnet-5-5`, dual review off. The settings live under `env` in `.claude/settings.json`; the plugin's README lists every variable. Project-specific rules go in `.claude/claude-security-guidance.md`, which the turn-end review reads. The plugin sends diffs and the files it reads to your model endpoint.

## Adding a hookify rule

Run `/hookify` right after an agent repeats a mistake you do not want to see again; with no argument it reads the conversation and proposes rules. Each rule is a markdown file with YAML frontmatter at `.claude/hookify.<name>.local.md`, matching a tool call by event and regex, then warning or blocking. A new rule takes effect on the next tool use, with no restart. `/hookify:list` shows the rules and `/hookify:configure` turns them on and off.

Good rules target a mistake that tooling cannot catch and that has a precise textual signature: a command, a path, a string in an edit. Serena's `remind` already covers built-in code reads, and `block-commit-on-main` covers commits on protected branches, so do not duplicate them.

## Secrets

Real values live in `.envrc` (gitignored, see `docs/agents/environment.md`) and never in `.mcp.json`, settings files or docs: reference them as `${VAR}`.
