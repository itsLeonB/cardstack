# Hooks, permissions and security

The Pi extensions in `.pi/extensions/` and the `@lystran/pi-serena-hooks` package enforce the rules below. Pi does not ask for approval before a tool call, so these extensions and the role files are the enforcement, not a settings allowlist.

## Enforced settings

- **Serena hooks** (`npm:@lystran/pi-serena-hooks`, declared in `.pi/settings.json`): `serena-hooks remind` nudges an agent that reads code with built-in tools; `activate`, `auto-approve` and `cleanup` manage the Serena session. See `docs/agents/conventions/serena.md`.
- **`.pi/extensions/block-commit-on-main.ts`**: refuses `git commit` and `git merge` while the session is on `main`.
- **`.pi/extensions/deny-big-output.ts`**: refuses `cat` or `git diff` output over 20 KB, and a `read` over 40 KB without `offset` or `limit`. The message names the cheaper route (`--stat`, `offset`/`limit`, `ctx_execute_file`). Prefix a bash command with `BIG_OK=1 ` to override; change thresholds with `DENY_BIG_BASH_MAX` and `DENY_BIG_READ_MAX` (bytes).

A tool an agent can reach, it can run. Scope comes from the role files under `docs/agents/roles/` and from the scope check in [`Delegating`](agents/workflows/implement.md#delegating), which inspects a worker's diff instead of restricting its tools. Project trust gates `.pi/settings.json`, `.pi/mcp.json`, `.pi/extensions/` and project `.agents/skills/`: grant it per folder with `/trust`, which is why a fresh checkout asks before loading them.

## Secrets

Real values live in `.envrc` (gitignored, see `docs/environment.md`). Never put them in `.mcp.json`, settings files or docs; reference them as `${VAR}`.
