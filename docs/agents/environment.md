# Environment setup

What a fresh clone needs before agents can work. MCP servers read `${VAR}` placeholders from `.mcp.json` out of the environment of the process that starts them, so the variables must be loaded before T3 Code Nightly starts.

1. Install `direnv` and hook it into your shell (`eval "$(direnv hook zsh)"` in `~/.zshrc`, or the bash or fish equivalent). Restart the shell.
2. Install Serena (`serena`, `serena-hooks`) and `jq`. Install the language servers with `scripts/lsp/install.sh`; `scripts/lsp/install.sh --check` only reports what is missing.
3. Copy the template and fill in your values: `cp .envrc.example .envrc`. `.envrc` is gitignored and holds secrets; never commit it.
4. Run `direnv allow` in the repo root. Run it again after every change to `.envrc`, because direnv refuses to load a changed file until you allow it.
5. Start T3 Code Nightly from that shell, inside the repo, so the app and every MCP server it launches inherit the variables. Start the orchestrator thread in `full-access` runtime mode (see `docs/agents/orchestration.md`).

Component worktrees do not hold gitignored files. `scripts/bootstrap-worktree/<component>.sh` copies `.envrc` from the main checkout into the worktree and runs `direnv allow` there, so a child thread runs it first.

When you add an MCP server that uses a new `${VAR}`, add the variable to `.envrc.example` in the same change, and tell the team to update their `.envrc` and run `direnv allow`.
