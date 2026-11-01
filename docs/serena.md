# Serena in pi

Setup notes for the maintainer, not agent instructions (agents follow `docs/agents/conventions/serena.md`). pi and its subagents use Serena's symbol-level tools as the primary way to read and edit code.

## Client wiring

| Piece | pi |
|---|---|
| MCP config | `~/.pi/agent/mcp.json` (per developer) |
| Serena launch | `--context ide`, `exposure: "deferred"` |
| Hooks | `.pi/settings.json`: `@lystran/pi-serena-hooks` |
| System prompt | `AGENTS.md` (no committed override) |
| Subagent prompts | not committed |

Two constraints shape the setup:

1. `serena-hooks` has no `pi` client ([hooks.py](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)). `@lystran/pi-serena-hooks` bridges the gap by mapping Serena's hook commands to pi's tool events.
2. Serena's MCP `instructions` field only says to call `initial_instructions`. The context prompt comes only from `print-system-prompt`. `pi-mcp-adapter` exposes server instructions through `mcp({ instructions: "serena" })` and never in the system prompt, so pi needs the prompt from `AGENTS.md` or `--append-system-prompt`.

Committed for pi: the `packages` entry in `.pi/settings.json`, the Serena section in `AGENTS.md`, and `docs/agents/conventions/serena.md`. The Serena MCP entry is not committed.

## Setup

### Add Serena per developer

Define Serena in your user config, never in the repo. pi merges project files over global ones, so a project entry overrides the global one and brings back the per-project trust gate. Every T3 worktree is a new project root, so the gate would block Serena in each.

In `~/.pi/agent/mcp.json`:

```json
{
  "mcpServers": {
    "serena": {
      "command": "serena",
      "args": ["start-mcp-server", "--context", "ide", "--project-from-cwd", "--open-web-dashboard", "false"],
      "exposure": "deferred"
    }
  }
}
```

- `ide` keeps `search_for_pattern`, which pi needs.
- `ide` sets `single_project: true`, which removes `activate_project`. The project is active at startup.
- `exposure: "deferred"` is the `mcp.json` form of `directTools: "search"`. Each tool becomes a direct tool the first time a search matches it or the proxy calls it ([tools.md](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)). The server connects lazily, see [Tool names and connecting](#tool-names-and-connecting). `mcp.json` ignores `directTools`, `lifecycle` and `idleTimeout`.
- Leave `cwd` unset. The adapter starts the server in the session's directory, which `--project-from-cwd` reads.
- Global servers skip the trust gate, so new worktrees need no approval.

### Bridge hooks with `@lystran/pi-serena-hooks`

Declare the bridge in `.pi/settings.json` so every pi session gets the same `activate`, `remind` and `cleanup` lifecycle:

```json
{
  "packages": ["npm:@lystran/pi-serena-hooks"]
}
```

`remind` runs before pi's `grep` and `multi_grep` calls, and before `bash` commands that start with `grep`, `rg`, `fgrep`, `egrep`, `ag` or `ack`. It injects Serena's reminder to use the symbolic tools. The package also fixes Serena's stale `activate_project` instruction, which matters because the `ide` context removes that tool ([pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)).

Each developer installs `pi-mcp-adapter` globally, puts `serena` and `serena-hooks` on `PATH`, and adds the Serena entry above.

### Load policy from `AGENTS.md`

pi loads `AGENTS.md` into every main session. Its Serena section holds the policy name, the tool path, and one trap (never call `activate_project` under a single-project context), and points to `docs/agents/conventions/serena.md` for the mapping. Keep the mapping out of `AGENTS.md`.

`--append-system-prompt "$(serena print-system-prompt --only-instructions)"` tracks Serena's own wording, but it's per-developer shell setup the repo can't commit. Treat it as a personal addition.

## Tool names and connecting

| | pi |
|---|---|
| Tool names | `serena_<tool>` in the `mcp` listing (unverified since the move to `exposure: "deferred"`); called through the proxy as `mcp({ server: "serena", tool: "<tool>", args: { ... } })` |
| When not listed | the server is lazy, so run `mcp({ connect: "serena" })` once per session |
| Project activation | already active; `activate_project` is absent |

In the ticket 06 session an agent looked for `mcp__serena__*` names on pi, found none, and nearly fell back to built-in file tools: pi's tools carry the `serena_` prefix, and `mcp({ })` showed Serena as "not connected yet" until `mcp({ connect: "serena" })` ran. If an agent reports that it cannot find Serena, check the server status with `mcp({ })` in pi, and connect it.

## Verify a context choice

Contexts decide the tool set, so check before you change one. This script runs Serena's MCP stdio handshake, sends `tools/list` and prints the tools each context exposes:

```bash
cd <repo> && python3 - <<'PY'
import json, subprocess
def probe(ctx):
    p = subprocess.Popen(["serena","start-mcp-server","--context",ctx,"--project-from-cwd"],
        stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, bufsize=1)
    p.stdin.write(json.dumps({"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}})+"\n")
    p.stdin.flush(); p.stdout.readline()
    p.stdin.write(json.dumps({"jsonrpc":"2.0","method":"notifications/initialized"})+"\n")
    p.stdin.write(json.dumps({"jsonrpc":"2.0","id":2,"method":"tools/list"})+"\n")
    p.stdin.flush()
    for line in p.stdout:
        try: msg = json.loads(line)
        except ValueError: continue
        if msg.get("id") == 2:
            p.terminate()
            return sorted(t["name"] for t in msg["result"]["tools"])
for ctx in ("ide",):
    names = probe(ctx)
    print(ctx, len(names), "search_for_pattern" in names, "activate_project" in names)
PY
```

Latest result on this repo:

| Context | Tools | `search_for_pattern` | `activate_project` |
|---|---|---|---|
| `ide` | 22 | present | absent |

## Sources

- [Serena configuration: contexts, modes, CLI arguments](https://oraios.github.io/serena/02-usage/050_configuration.html)
- [Serena: connecting your MCP client](https://oraios.github.io/serena/02-usage/030_clients.html)
- [Serena `hooks.py`: hook clients and reminder rules](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)
- [Serena issue: missing `serena-hooks` client support](https://github.com/oraios/serena/issues/1869)
- [pi-mcp-adapter: configuration, precedence and trust gate](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/configuration.md)
- [pi-mcp-adapter: direct tools and subagent integration](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)
- [pi-mcp-adapter on pi.dev](https://pi.dev/packages/pi-mcp-adapter)
- [@lystran/pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)
- [pi: MCP servers](https://pi.dev/docs/latest/mcp)
