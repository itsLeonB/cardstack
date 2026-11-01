# pi-mcp-adapter: global vs project server definitions

TL;DR: Lazy/progressive discovery is a per-server adapter feature, not tied to config location, so a global server stays lazy — but `directTools: "search"` is silently ignored inside Pi's own `~/.pi/agent/mcp.json`, so put it in `~/.pi/agent/mcp-adapter.json` (or `~/.config/mcp/mcp.json`), or use Pi's `"exposure": "deferred"`.
Global servers are exempt from the project-trust/approval prompt; only servers defined by `.mcp.json`, `.pi/mcp.json`, `.pi/mcp-adapter.json` (and ancestors/imports from them) are gated, which is why `serena` shows "blocked by project trust".
Same-name servers across files merge field by field (project wins); a global `serena` using `--project-from-cwd` works because stdio servers spawn with the Pi session cwd.

Installed copy checked: `pi-mcp-adapter` 7.0.0 at `~/.pi/agent/npm/node_modules/pi-mcp-adapter` (`package.json`). Paths below are relative to that dir unless absolute. Upstream: https://github.com/nicobailon/pi-mcp-adapter

## 1. Config locations and precedence

- Sources, lowest to highest: `~/.config/mcp/mcp.json`, `~/.agents/mcp.json`, `~/.agents/mcp/mcp.json`, `<Pi agent dir>/mcp.json`, `<Pi agent dir>/mcp-adapter.json`, opted-in ancestors, `.mcp.json`, `.pi/mcp.json`, `.pi/mcp-adapter.json` (`docs/configuration.md:73-85`).
- Global = `~/.pi/agent/mcp.json` (Pi format, read on Pi ≥ 0.99, `docs/configuration.md:41`) and `~/.pi/agent/mcp-adapter.json` (adapter format, `docs/configuration.md:26`). Project = `.mcp.json`, `.pi/mcp.json`, `.pi/mcp-adapter.json`.
- Ancestor discovery is off by default and can only be enabled from a user-global source (`docs/configuration.md:87`).
- Local state: `~/.pi/agent/mcp.json:2-` defines only `radius` and `context-mode` (secrets-free). Repo `.mcp.json` defines `postgres`, `serena`, `context7`, `postgres-prod`. Repo `.pi/mcp-adapter.json` defines `serena` with `directTools: "search"`. No `~/.pi/agent/mcp-adapter.json` exists.

## 2. Settings controlling lazy / progressive discovery

- Default is already lazy and context-cheap: one `mcp` proxy tool (~200 tokens), servers connect on first call, tool metadata cached on disk so search/list work offline (`README.md` "How It Works"; https://github.com/nicobailon/pi-mcp-adapter#how-it-works).
- Per-server `lifecycle`: `lazy` (default), `eager`, `keep-alive`, `lazy-keep-alive`; `idleTimeout` (default 10 min) stops idle stdio servers (`docs/configuration.md:130-160`).
- Metadata cache is global, not per-project: `~/.pi/agent/mcp-cache.json` (`docs/tools.md:181`). Direct tools register from this cache with no server connection at startup.
- Per-server `directTools`: `false` (default), `true`, `string[]`, or `"search"`; per-server overrides global `settings.directTools` (`docs/tools.md:75-121`, `docs/configuration.md:214`).
- `"search"` = tools registered as real tools with real schemas but inactive; `mcp({ search })`, a successful `mcp({ tool })`, or Pi's `tool_search` activates matches additively (`docs/tools.md:123-143`). On Pi ≥ 0.99 they are Pi deferred tools grouped under `mcp__<server>` (`docs/tools.md:141-143`).
- Even without `directTools`, every cached tool is a Pi deferred tool `mcp__<server>__<tool>` reachable via `tool_search`/codemode (`docs/tools.md:199-204`).
- Setting-location caveat: Pi's own `mcp.json` is parsed by `translatePiMcpServer`, which destructures only `type, command, args, env, cwd, url, headers, oauth, exposure, toolExposure, enabled, timeout, auth, description`; any other key (including `directTools`, `lifecycle`, `idleTimeout`) goes to `ignored` (`config.ts:1387-1400`). Reported at startup and in `/mcp-adapter` (`docs/configuration.md:57`).
- Pi-format equivalents: `"exposure": "deferred"` → `directTools: "search"`, `"direct"` → `true`, `toolExposure` names set to `"direct"` → `directTools: [names]` (`config.ts:1437-1453`, `docs/configuration.md:52-53`).
- So for a global server: use `~/.pi/agent/mcp-adapter.json`, `~/.config/mcp/mcp.json`, or `"exposure": "deferred"` in `~/.pi/agent/mcp.json`. `lifecycle`/`idleTimeout` have no Pi-format equivalent, so they need an adapter-format file (inference from the destructure above plus `docs/configuration.md:26,41`).
- Adapter fields work identically from global and project adapter-format files; the only difference is the project trust gate below.

## 3. Project trust / approval

- Servers defined or changed by project-scoped files (`.mcp.json`, `.pi/mcp-adapter.json`, `.pi/mcp.json`, opted-in ancestors, project imports/plugins) do not run until Pi trusts the project and the user approves each server (`docs/configuration.md:116`, `:59`).
- Untrusted project → "blocked by project trust" (`project-server-trust.ts:42`; status line `init.ts:963`). Trusted but unapproved → prompt in interactive mode, skipped in print/JSON/RPC (`docs/configuration.md:116,120`).
- Approval store: `~/.pi/agent/mcp-project-approvals.json` (`project-server-trust.ts:12`), record = `projectRoot` + `serverName` + `definitionHash` + `approvedAt`; any definition change needs re-approval; git worktrees of one repo share approvals (`docs/configuration.md:116`).
- Local store has approvals for this repo's `postgres`, `context7`, `serena`, `postgres-prod` (hashes in `~/.pi/agent/mcp-project-approvals.json`). `serena`'s hash dates 2026-10-10; editing its definition (e.g. adding `directTools`) changes the hash and needs re-approval.
- Global servers are exempt: sources with `scope: "global"` are never added to `projectServers`; only `scope: "project"` sources are (`config.ts:628,651,687`, source scopes at `config.ts:849-901` vs `934-977`). The doc states the gate applies to project-scoped files only and that global/import/plugin servers keep existing behavior (`docs/configuration.md:116,120`).
- Escape hatches (all user-global only): `settings.projectServers: "allow"` for headless; `pi --mcp-project-servers allow` per process (`docs/configuration.md:120-122`).
- Pi's own project trust is separate: `~/.pi/agent/trust.json` lists trusted roots (`/home/leon/Projects/itsLeonB` is `true`, which covers this repo).

## 4. Same server name in global and project

- No duplicate error. Later (higher-precedence) source wins, and entries merge field by field (`docs/configuration.md:97`; `config.ts:1050-1125` `mergeServerMaps`).
- Transport switch guard: a later entry with `command` drops inherited `url`/auth fields, a later `url` drops inherited `command`/`args`/`env`/`cwd` (`config.ts:1100-1125`).
- Credentials are bound to the `url` they were written for; a changed `url` drops the lower file's credentials with a startup notice (`docs/configuration.md:97`).
- Exception: within Pi's own files, `.pi/mcp.json` replaces the same-named `<agent dir>/mcp.json` server as a whole (`docs/configuration.md:59`).
- Provenance for a merged server is the last (highest) file that defined it; if that file is project-scoped, the whole server counts as a project server (`config.ts:665` comment "the file that set it last decides its scope"; `config.ts:1050`).
- Consequence: defining `serena` globally and keeping a project `.pi/mcp-adapter.json` override still works, but the merged server is gated by project trust because the project file sets it last. To make it fully global-exempt, remove it from all project files.
- This repo already has `serena` in both `.mcp.json` (`type, command, args`) and `.pi/mcp-adapter.json` (adds `directTools`, `description`); they merge today.

## 5. `--project-from-cwd` stdio server defined globally

- Stdio servers spawn with `cwd = definition.cwd ?? McpServerManager.defaultCwd`, where `defaultCwd` is the Pi session `ctx.cwd` (`server-manager.ts:1214`; `init.ts:137,174`). A global definition without `cwd` therefore spawns in the project you opened Pi in, so `--project-from-cwd` resolves to the right project.
- Gotcha 1: never set `cwd` on the global entry, or Serena activates that fixed dir. Configured `cwd` that does not exist/isn't a directory throws (`server-manager.ts:1218-1219`).
- Gotcha 2: the process is spawned at connect time and (with lazy lifecycle) torn down after the idle timeout, so cwd is the cwd of the session that last connected it. Each Pi session runs its own server processes (`README.md` Limitations).
- Gotcha 3: the adapter also sends the session dir as an MCP root unless `roots: false` (`server-manager.ts:1529-1532`, `docs/servers.md:62`); harmless for Serena but worth knowing.
- Gotcha 4: the metadata cache `~/.pi/agent/mcp-cache.json` is shared, and a server whose definition is unchanged reuses cached tools across projects; Serena's tool list can differ per project/context, so a stale list is possible until `/mcp-adapter reconnect serena` (inference from `docs/tools.md:181`, not tested).
- Gotcha 5: first-ever session for a new/changed server connects it once to populate the cache, so tools fall back to proxy-only until the cache fills (`docs/tools.md:181`).

## Recommendation (if moving serena global)

- Put it in `~/.pi/agent/mcp-adapter.json` with `command`, `args` (incl. `--project-from-cwd`), `directTools: "search"`, no `cwd`.
- Delete `serena` from repo `.mcp.json` and `.pi/mcp-adapter.json`, or the project copy still wins and keeps the trust gate.
- Not tested: nothing was run or modified; claims come from docs and source only.
