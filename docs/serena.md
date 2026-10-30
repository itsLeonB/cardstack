# Serena across Claude Code and pi

Setup notes for the maintainer, not agent instructions (agents follow `docs/agents/conventions/serena.md`). Both clients, and pi's subagents, use Serena's symbol-level tools as the primary way to read and edit code.

## Client wiring

| Piece | Claude Code | pi |
|---|---|---|
| MCP config | `.mcp.json` (native) | `.mcp.json` via `pi-mcp-adapter` |
| Serena launch | `.mcp.json`, `--context claude-code` | `.pi/mcp-adapter.json`, `--context ide`, `directTools: "search"` |
| Hooks | `.claude/settings.json`: `serena-hooks activate/remind/auto-approve/cleanup --client=claude-code` | `.pi/settings.json`: `@lystran/pi-serena-hooks`, same commands in the `claude-code` hook format |
| System prompt | `--system-prompt "$(serena print-system-prompt --context claude-code)"` per invocation | `AGENTS.md` (no committed override) |
| Subagent prompts | `.claude/agents/*.md` | not committed |

Two constraints shape the setup:

1. `serena-hooks` has no `pi` client. Its `--client` values are `claude-code`, `codebuddy`, `vscode`, `codex` and `grok` ([hooks.py](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)). `@lystran/pi-serena-hooks` bridges the gap: it speaks the `claude-code` hook JSON and maps it to pi's tool events.
2. Serena's MCP `instructions` field only says to call `initial_instructions`. The context prompt comes only from `print-system-prompt`. `pi-mcp-adapter` exposes server instructions through `mcp({ instructions: "serena" })` and never in the system prompt, so pi needs the prompt from `AGENTS.md` or `--append-system-prompt`.

Committed for pi: `.pi/mcp-adapter.json`, the `packages` entry in `.pi/settings.json`, the Serena section in `AGENTS.md`, and `docs/agents/conventions/serena.md`. `.mcp.json` passes `--open-web-dashboard false` so neither client opens the dashboard on launch.

## Setup

### Share `.mcp.json`

Claude Code reads `.mcp.json` natively. `pi-mcp-adapter` reads it as project shared config, layer 7 of its precedence list ([configuration.md](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/configuration.md)). Keep the `serena` entry on the `claude-code` context with `--project-from-cwd` and `--open-web-dashboard false`, so Claude Code keeps its `structured_tool_output: false` workaround.

### Set pi's context in `.pi/mcp-adapter.json`

The `claude-code` context excludes `search_for_pattern`, which pi needs, and works around a Claude Code structured-output bug that doesn't affect pi. A nearer adapter layer replaces a same-named server wholesale, so pi gets the `ide` context and Claude Code is untouched.

```json
{
  "mcpServers": {
    "serena": {
      "command": "serena",
      "args": ["start-mcp-server", "--context", "ide", "--project-from-cwd", "--open-web-dashboard", "false"],
      "directTools": "search"
    }
  }
}
```

- `directTools: "search"` registers Serena's tools as deferred tools in the `mcp__serena` namespace: `tool_search` loads them on demand and the `mcp` proxy stays available ([tools.md](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)). Use `true` to load all 22 tools every turn and skip discovery.
- The `ide` context sets `single_project: true`, which removes `activate_project`; the project is active from startup.
- Project servers need one interactive approval per effective definition. Headless sessions skip them unless the user-global adapter settings set `projectServers: "allow"`.

### Bridge hooks with `@lystran/pi-serena-hooks`

Declare the bridge in `.pi/settings.json` so every pi session gets the same `activate`, `remind` and `cleanup` lifecycle as Claude Code:

```json
{
  "packages": ["npm:@lystran/pi-serena-hooks"]
}
```

`remind` runs before pi's `grep` and `multi_grep` calls, and before `bash` commands that start with `grep`, `rg`, `fgrep`, `egrep`, `ag` or `ack`. It injects Serena's reminder to use the symbolic tools. The package also fixes Serena's stale `activate_project` instruction, which matters because the `ide` context removes that tool ([pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)).

Each developer installs `pi-mcp-adapter` globally and puts `serena` and `serena-hooks` on `PATH`.

### Load policy from `AGENTS.md`

pi loads `AGENTS.md` into every main session. Its Serena section holds the policy name, the tool path, and one trap (never call `activate_project` under a single-project context), and points to `docs/agents/conventions/serena.md` for the mapping. Keep the mapping out of `AGENTS.md`.

`--append-system-prompt "$(serena print-system-prompt --only-instructions)"` tracks Serena's own wording, but it's per-developer shell setup the repo can't commit. Treat it as a personal addition.

### Share policy with subagents

The tool mapping and pre-edit workflow live only in `docs/agents/conventions/serena.md`. Both agent files reference it next to `general.md` and their component conventions, so a child reads it before its first code edit. The doc states pi's tool path: the `mcp` proxy, or `mcp__serena__*` after `tool_search` activates them.

How subagents reach Serena ([pi-subagents agents.md](https://github.com/nicobailon/pi-subagents/blob/main/docs/agents.md)):

- A child with `tools` omitted gets pi's builtin tools. A background child also loads ambient extensions, including the `mcp` proxy. A local foreground child loads none, so it can't reach Serena.
- `mcp: serena` in frontmatter resolves the adapter's direct tools, in a background child only.
- `extensions:` in frontmatter disables ambient discovery for that agent, which removes the proxy.
- `inheritProjectContext` is off by default for custom agents, so a child doesn't see `AGENTS.md` unless the agent opts in.

Run Serena-using subagents as background children (pi-subagents' default) and put the policy in the agent prompt, not in `AGENTS.md` inheritance.

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
for ctx in ("ide","claude-code"):
    names = probe(ctx)
    print(ctx, len(names), "search_for_pattern" in names, "activate_project" in names)
PY
```

Latest result on this repo:

| Context | Tools | `search_for_pattern` | `activate_project` |
|---|---|---|---|
| `ide` | 22 | present | absent |
| `claude-code` | 21 | absent | absent |

## Sources

- [Serena configuration: contexts, modes, CLI arguments](https://oraios.github.io/serena/02-usage/050_configuration.html)
- [Serena: connecting your MCP client](https://oraios.github.io/serena/02-usage/030_clients.html)
- [Serena `hooks.py`: hook clients and reminder rules](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)
- [Serena issue: missing `serena-hooks` client support](https://github.com/oraios/serena/issues/1869)
- [pi-mcp-adapter: configuration and precedence](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/configuration.md)
- [pi-mcp-adapter: direct tools and subagent integration](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)
- [pi-mcp-adapter on pi.dev](https://pi.dev/packages/pi-mcp-adapter)
- [@lystran/pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)
- [pi-subagents: agents, tool allowlists, context inheritance](https://github.com/nicobailon/pi-subagents/blob/main/docs/agents.md)
- [pi: MCP servers](https://pi.dev/docs/latest/mcp)
