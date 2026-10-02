# Serena across Claude Code and pi

Setup notes for the human maintainer, not agent instructions. Agents follow `docs/agents/conventions/serena.md`.

How Serena is wired so both clients, and pi's subagents, use Serena's symbol-level tools as the primary way to read and edit code in this repo.

## What each client reads today

| Piece | Claude Code | pi |
|---|---|---|
| MCP server config | `.mcp.json` (native) | `.mcp.json` via the `pi-mcp-adapter` extension |
| Serena launch | `.mcp.json` → `--context claude-code` | `.pi/mcp-adapter.json` → `--context ide`, `directTools: "search"` |
| Lifecycle hooks | `.claude/settings.json` → `serena-hooks activate/remind/auto-approve/cleanup --client=claude-code` | `.pi/settings.json` → `@lystran/pi-serena-hooks`, same commands in Serena's `claude-code` hook format |
| System prompt | `--system-prompt "$(serena print-system-prompt --context claude-code)"` per invocation | `AGENTS.md` (pi has no committed system-prompt override) |
| Subagent prompts | `.claude/agents/*.md`, symlinked as `.pi/agents/*.md` | same files |

Two facts drive the recommendations below.

1. `serena-hooks` has no `pi` client. Its `--client` enum is `claude-code`, `codebuddy`, `vscode`, `codex`, `grok` ([hooks.py](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)). The community package `@lystran/pi-serena-hooks` bridges this by speaking Serena's `claude-code` hook JSON format and mapping it onto pi's tool events.
2. Serena's MCP `instructions` field only carries the "call `initial_instructions`" nudge, not the context prompt. The context prompt reaches a model only through `print-system-prompt`. pi-mcp-adapter surfaces server instructions through `mcp({ instructions: "serena" })`, never into the system prompt, so for pi the prompt has to come from `AGENTS.md` or `--append-system-prompt`.

## Setup

Committed for pi: `.pi/mcp-adapter.json`, the `packages` entry in `.pi/settings.json`, the Serena section in `AGENTS.md`, and `docs/agents/conventions/serena.md`. Claude Code's `.mcp.json` and `.claude/settings.json` are unchanged.

### 1. `.mcp.json` stays the single shared source

Claude Code reads it natively; pi-mcp-adapter reads it as project shared config, layer 7 of its precedence list, below `.pi/mcp-adapter.json` ([configuration.md](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/configuration.md)). Leave the existing `serena` entry alone so Claude Code keeps its `claude-code` context and its `structured_tool_output: false` workaround.

### 2. pi's Serena context: `.pi/mcp-adapter.json`

The `claude-code` context excludes `search_for_pattern`, which pi wants, and exists to work around a Claude Code structured-output bug that does not apply to pi. A nearer adapter layer replaces the same-named server wholesale, so pi gets the `ide` context without touching Claude Code.

```json
{
  "mcpServers": {
    "serena": {
      "command": "serena",
      "args": ["start-mcp-server", "--context", "ide", "--project-from-cwd"],
      "directTools": "search"
    }
  }
}
```

`directTools: "search"` registers Serena's tools as pi deferred tools in the `mcp__serena` namespace, so `tool_search` loads them on demand and the `mcp` proxy stays available ([tools.md](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)). The `ide` context keeps `single_project: true`, so `activate_project` is removed and the project is active from startup. Use `directTools: true` instead if you would rather pay for 22 tools of system prompt on every turn in exchange for zero discovery cost.

Project servers need one interactive approval per effective definition, and they are skipped in headless sessions unless the user-global adapter settings set `projectServers: "allow"`.

### 3. pi's hooks bridge: `@lystran/pi-serena-hooks`

The bridge is declared in project settings, so every pi session in this repo gets the same `activate` / `remind` / `cleanup` lifecycle Claude Code devs get:

```json
{
  "packages": ["npm:@lystran/pi-serena-hooks"]
}
```

`remind` fires before pi's `grep` and `multi_grep` calls and before `bash` commands that start with `grep`, `rg`, `fgrep`, `egrep`, `ag`, or `ack`, and injects Serena's "use the symbolic tools" reminder. The plugin forward-fixes Serena's stale `activate_project` instruction, which is necessary here because the `ide` context removes that tool ([pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)).

Prerequisites stay per-developer: `pi-mcp-adapter` installed globally, and `serena` plus `serena-hooks` on `PATH`.

### 4. Drive the main pi context from `AGENTS.md`

pi loads `AGENTS.md` into every main session, so that section carries the policy name, the tool path, and the one trap — never call `activate_project` under a single-project context — and points at `docs/agents/conventions/serena.md` for the full mapping. The mapping itself does not belong in `AGENTS.md`.

`--append-system-prompt "$(serena print-system-prompt --only-instructions)"` gives the same effect per invocation and picks up Serena's own wording as it evolves, but it is per-developer shell setup, not something the repo can commit. `AGENTS.md` is the committable path; the flag is a reasonable personal addition on top.

### 5. Drive subagents from one shared policy

`.pi/agents/backend-agent.md` and `.pi/agents/frontend-agent.md` are symlinks to the Claude Code agent files, so one edit covers both clients.

Previously the mapping was duplicated inline in both agent files;
The tool-selection mapping and the pre-edit workflow now live only in `docs/agents/conventions/serena.md`; both agent files reference it alongside `general.md` and their component conventions file, so the child has to read it before the first code edit. The prompts no longer call themselves Claude Code, and the shared doc states pi's tool path explicitly: the `mcp` proxy, or `mcp__serena__*` once `tool_search` activates them.

Subagents reach Serena as follows, from [pi-subagents agents.md](https://github.com/nicobailon/pi-subagents/blob/main/docs/agents.md):

- A child with `tools` omitted gets pi's builtin tools, and a background child also loads ambient extensions, which is where the `mcp` proxy lives. Local foreground children load no ambient extensions and therefore have no Serena access at all.
- `mcp: serena` in frontmatter resolves the adapter's direct tools, but only in a background child, because the adapter has to be loaded to resolve it.
- `extensions:` in frontmatter disables ambient discovery for that agent, which would remove the proxy again.
- `inheritProjectContext` defaults to off for custom agents, so a child does not see `AGENTS.md` unless the agent opts in.

Practical consequence: run Serena-using subagents as background children, which is pi-subagents' default, and rely on the agent prompt rather than `AGENTS.md` inheritance for the policy.

## Verifying a context choice

Contexts decide the tool set, so check one before changing it. This drives Serena's MCP stdio handshake, sends `tools/list`, and prints which tools the context exposes:

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

Latest result on this repo: `ide` 22 tools, `search_for_pattern` present, `activate_project` absent; `claude-code` 21 tools, `search_for_pattern` absent, `activate_project` absent.

## Sources

- [Serena configuration: contexts, modes, CLI arguments](https://oraios.github.io/serena/02-usage/050_configuration.html)
- [Serena: connecting your MCP client](https://oraios.github.io/serena/02-usage/030_clients.html)
- [Serena `hooks.py` hook clients and reminder rules](https://github.com/oraios/serena/blob/main/src/serena/hooks.py)
- [Serena issue: missing `serena-hooks` client support](https://github.com/oraios/serena/issues/1869)
- [pi-mcp-adapter configuration and precedence](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/configuration.md)
- [pi-mcp-adapter direct tools and subagent integration](https://github.com/nicobailon/pi-mcp-adapter/blob/main/docs/tools.md)
- [pi-mcp-adapter on pi.dev](https://pi.dev/packages/pi-mcp-adapter)
- [@lystran/pi-serena-hooks](https://pi.dev/packages/@lystran/pi-serena-hooks)
- [pi-subagents: agents, tool allowlists, context inheritance](https://github.com/nicobailon/pi-subagents/blob/main/docs/agents.md)
- [pi: MCP servers](https://pi.dev/docs/latest/mcp)
