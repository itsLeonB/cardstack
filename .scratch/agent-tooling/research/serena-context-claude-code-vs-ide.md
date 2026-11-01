# Serena `--context claude-code` vs `--context ide`

TL;DR: both are single-project contexts that exclude the same five tools (`create_text_file`, `read_file`, `execute_shell_command`, `find_file`, `list_dir`); `claude-code` also excludes `search_for_pattern` and sets `structured_tool_output: false`.
TL;DR: the real difference is the prompt: `claude-code` is a long, Claude-Code-specific order to never use Read/Edit on code; `ide` is a short, harness-neutral "prefer Serena, avoid reading whole files".
TL;DR: for Pi (built-in `read`/`edit`/`bash`/`write`, no deferred tools) `ide` fits better; a custom context copying `claude-code` with Pi's tool names fits best if a hard "Serena first" rule is wanted (inference, not tested).

Sources: `main` at `ede329aaf0c824bea75455a2926f01420d7aa620`, context files last touched in `84947c76778946d8823f53dd7a2d7d2ca4d0db8c`, latest release `v2.1.0` (2026-10-19), checked 2026-11-01 via GitHub API: https://api.github.com/repos/oraios/serena/commits/main and https://api.github.com/repos/oraios/serena/releases/latest.
The locally installed contexts (`~/.local/share/uv/tools/serena-agent/.../serena/resources/config/contexts/`) are byte-identical to `main` for both files (diffed), although `serena --version` prints `1.7.0`.
Context files: https://github.com/oraios/serena/blob/main/src/serena/resources/config/contexts/claude-code.yml and https://github.com/oraios/serena/blob/main/src/serena/resources/config/contexts/ide.yml.

## 1. Tools

| Setting | `claude-code` | `ide` |
|---|---|---|
| `excluded_tools` | `create_text_file`, `read_file`, `execute_shell_command`, `find_file`, `list_dir`, `search_for_pattern` | `create_text_file`, `read_file`, `execute_shell_command`, `find_file`, `list_dir` |
| `single_project` | `true` | `true` |
| `structured_tool_output` | `false` (comment: Claude Code does not unpack structured tool output, causing a lot of escaping) | not set, inherits the template default `null` (auto-detect) |
| `included_optional_tools`, `fixed_tools`, `tool_description_overrides` | not set / `{}` | not set / `{}` |

- Neither file lists `activate_project`; `single_project: true` disables it when a project is given at startup (comment in both YAML files, plus docs: https://oraios.github.io/serena/02-usage/050_configuration.html#contexts, source `docs/02-usage/050_configuration.md` lines 149-153).
- With a startup project, the tool set is further trimmed to what the project's config needs (same docs paragraph).
- So `ide` keeps `search_for_pattern` (regex search, also for non-code files) and `claude-code` drops it, expecting the client's Grep instead.
- Everything else (symbol overview/search, references, `replace_symbol_body`, `insert_*`, `replace_content`, rename, memory tools, etc.) is available in both.
- `serena start-mcp-server --help` (local CLI): `--context TEXT` is a built-in name or a path to a custom YAML, default `desktop-app`; also `--project`, `--project-from-cwd`, `--mode`, `--add-mode`.
- `--project-from-cwd` is documented as intended for CLI agents like Claude Code, Gemini and Codex (https://oraios.github.io/serena/02-usage/020_running.html).

## 2. System prompt / instructions

- A context's `prompt` field is added to the system prompt / initial instructions of agents started in that context (https://github.com/oraios/serena/blob/main/src/serena/resources/config/contexts/context.template.yml; docs `050_configuration.md` line 229).
- `ide` prompt: file ops, basic line edits and shell are handled by the agent's own tools; prioritize Serena's tools; avoid reading whole source files unless strictly necessary; use overview and symbolic search for code; use the pattern search tool for non-code files or when the symbol name path is unknown.
- `claude-code` prompt is much longer and Claude-Code-specific:
  - it tells the model to load Serena's tools immediately if deferred, and to use "the tool search tool" and read each tool's schema before use;
  - it says to always edit via Serena because "your built-in tools will deny such edits" (Claude Code's read-before-edit guard);
  - it has a table: `Read` forbidden for discovery, `Glob` and `Grep` allowed for discovery only, `Edit` forbidden (use `replace_symbol_body` / `insert_*_symbol` / `replace_content`);
  - it lists "disallowed reasoning" excuses for using Read/Edit.
- The `ide` prompt never forbids the client's read/edit; the `claude-code` prompt forbids them for code files.

## 3. Intended use

- `claude-code`: "Optimized for use with Claude Code, it disables tools that would duplicate Claude Code's built-in capabilities" (docs `050_configuration.md` line 138); YAML description: "Claude Code (CLI agent where file operations, basic edits, etc. are already covered; single project mode)".
- `ide`: "Generic context for IDE assistants/coding agents, e.g. VSCode, Cursor, or Cline, focusing on augmenting existing capabilities. Basic file operations and shell execution are assumed to be handled by the assistant's own capabilities" (docs line 141-142).
- Docs recommend `--project <path>` with a single-project context such as `ide` or `claude-code` for per-workspace clients (https://oraios.github.io/serena/02-usage/030_clients.html, source `docs/02-usage/030_clients.md` lines 26-30); `ide`, `claude-code` and `grok` are the three single-project contexts (`050_configuration.md` line 149).
- Claude Code setup in the docs is `serena start-mcp-server --context claude-code --project-from-cwd` (global) or `--project "$(pwd)"` (per project), plus optional `serena-hooks` (`030_clients.md` lines 107-121).

## 4. Pi harness and a Serena-first harness

- Pi is not a documented Serena client; there is no Pi-specific context in `contexts/` (listing: `agent`, `antigravity`, `chatgpt`, `claude-code`, `codebuddy`, `codex`, `copilot-cli`, `desktop-app`, `grok`, `ide`, `jb-ai-assistant`, `jb-copilot-plugin`, `junie`, `oaicompat-agent`, `vscode`, `zcode`; https://api.github.com/repos/oraios/serena/contents/src/serena/resources/config/contexts).
- Tool-set impact is small either way: both contexts assume the client has its own read/write/shell, which Pi does (`read`, `edit`, `write`, `bash`).
- Pi has no dedicated grep/find/ls tool, so `claude-code` (which drops `search_for_pattern` and relies on the client's Grep/Glob) leaves pattern search to `bash` (`rg`); `ide` keeps `search_for_pattern` as a Serena-side option (inference from the exclusion lists above).
- The `claude-code` prompt names Claude Code mechanics (`Read`/`Glob`/`Grep`/`Edit` names, deferred tools, a tool-search tool, the read-before-edit denial); in Pi these are partly wrong (Pi's tool names are lowercase `read`/`edit`, and Pi's built-in edit has no such denial), so the instructions are noisier but their intent (Serena first, no Read/Edit on code) still matches a Serena-first harness (inference).
- `ide` prompt is neutral and fits Pi as-is, but is softer: it asks to prioritize Serena and avoid whole-file reads, with no "forbidden" rule; a harness that wants Serena strictly preferred must add that rule in its own instructions (here: `AGENTS.md` and `docs/agents/conventions/serena.md`).
- Option for a strict Pi setup: `serena context create pi` (or `--context /path/to/pi.yml`) starting from `claude-code.yml`, rewriting the prompt table with Pi's tool names and dropping the Claude-Code-only lines (docs: `serena context create|edit|list`, `050_configuration.md` lines 160-166); also set `structured_tool_output` deliberately, since `false` is a Claude Code bug workaround and Pi may handle structured output (unverified).
- `serena-hooks` (`remind`, `auto-approve`, `activate`, `cleanup`) are wired through Claude Code's hook settings (`--client=claude-code`; docs also show `--client=codex`) and are not documented for Pi, so the drift-reminder behavior the `claude-code` prompt compensates for is not available there unless Pi gets an equivalent (`030_clients.md` lines 129-195).
- This repo's `.mcp.json` starts Serena with `--context claude-code --project-from-cwd`; any Pi session reading that file inherits the Claude-Code prompt and the `search_for_pattern` exclusion.

## Not verified

- No live run of the Serena server in either context; findings come from the YAML, docs and local diff.
- How Pi's MCP adapter surfaces the context `prompt` (initial instructions vs. a tool) was not checked.
- Tool-reference page (`docs/01-about/035_tools.md`) was not parsed for per-tool descriptions.
