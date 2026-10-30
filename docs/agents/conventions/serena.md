# Serena tool policy

Serena is an MCP server that exposes semantic, symbol-aware tools for reading, searching, and editing code. For every agent its tools are the PRIMARY tools for code work in this project. Built-in file tools are SECONDARY and must not be used on code files when a Serena equivalent exists.

## Who uses Serena

Everyone uses Serena: the orchestrator, the component agents and the advisor. Read this file before your first tool call on a code file.

- The orchestrator's Serena is rooted at the main checkout.
- A component agent works in its own worktree, and its Serena is rooted at that worktree, so a symbol read or edit lands in that worktree and not in the main checkout.
- The advisor uses Serena for reads in the main checkout, and reads a child's worktree by absolute path with the built-in tools, because its Serena is rooted at the main checkout.

Research behind this: `.scratch/agent-tooling/research/serena-per-worktree-subagents.md`.

## Reaching the tools

- Serena's tools have Serena's own names (`initial_instructions`, `get_symbols_overview`, `find_symbol`, `replace_symbol_body`, ...). A harness adds its own prefix or namespace to them, so find each tool by its Serena name whatever the prefix, in your tool list or with the harness's tool search. If the harness puts MCP tools behind a proxy tool, call Serena's through it by the Serena name.
- If no Serena tool is listed, the server is not connected yet. Connect or enable it through the harness's own MCP mechanism (its connect command or MCP settings) and look again. Fall back to the built-in tools only after that fails, and say so in your report.
- Before your first code tool call, call `initial_instructions`.
- Call the project-activation tool (`activate_project`) only if it exists. Under a single-project setup the project is already active and the tool is absent.

## Mapping (use the right column, not the left)

Task                                    Tool to use
--------------------------------------  ----------------------------------------
See a code file's structure             get_symbols_overview
Read a specific symbol's body           find_symbol (include_body=true)
Find a symbol by name across the repo   find_symbol
Find references / callers               find_referencing_symbols
Find declarations / implementations     find_declaration / find_implementations
Edit a symbol's body                    replace_symbol_body
Insert near a symbol                    insert_before_symbol / insert_after_symbol
Pattern replace inside a file           replace_content
Rename / move / delete a symbol         rename_symbol / safe_delete_symbol
Diagnostics for a file or symbol        get_diagnostics_for_file

In Go, a method is a top-level symbol, not a child of its receiver type: `find_symbol` with `ClerkKeys/FindKey` returns nothing and `ClerkVerifier` at depth 1 lists no methods, so search the bare method name (`FindKey`) and narrow with `relative_path`. A `cat`, `head`, `tail`, `nl` or `sed -n` on a `.go`, `.ts` or `.tsx` file through Bash counts as a built-in read, and Serena's `serena-hooks remind` PreToolUse hook nudges you after a streak of them.

Built-in read/edit/glob/grep are permitted on code files ONLY when:

- Serena has been tried on the target and failed.
- The file is not parseable as code.
- You need a regex search across many files that the symbolic tools cannot express. Grep is acceptable as a discovery step, but follow-up reads and edits on matched code files still go through Serena.
- You need a few lines and a symbolic read would be overkill.
- You have to read the whole file for some reason.

Built-in read/edit/glob are fine for non-code files: markdown, JSON, YAML, TOML, `.env`, config files, lockfiles, `go.mod`/`go.sum`, `package.json`, plain text, images.

## Required workflow before editing code

1. `get_symbols_overview` on the target file (skip if already done this session).
2. `find_symbol` with `include_body=true` for the specific symbols you will touch.
3. Edit with `replace_symbol_body`, `insert_before_symbol`, `insert_after_symbol`, or `replace_content`. Never use the built-in edit tool on a code file when one of these fits.
