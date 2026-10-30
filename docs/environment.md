# Environment setup

What a fresh clone needs before agents can work. MCP servers read `${VAR}` placeholders from `.mcp.json` out of the environment of the process that starts them, so the variables must be loaded before your agent starts.

1. Install `direnv` and hook it into your shell (`eval "$(direnv hook zsh)"` in `~/.zshrc`, or the bash or fish equivalent). Restart the shell.
2. Install Serena (`serena`, `serena-hooks`) and `jq`. Install the language servers with `scripts/lsp/install.sh`; `scripts/lsp/install.sh --check` only reports what is missing.
3. Copy the template and fill in your values: `cp .envrc.example .envrc`. `.envrc` is gitignored and holds secrets; never commit it.
4. Run `direnv allow` in the repo root. Run it again after every change to `.envrc`, because direnv refuses to load a changed file until you allow it.
5. Start your agent from that shell, inside the repo, so it and every MCP server it launches inherit the variables.

Component worktrees do not hold gitignored files. `scripts/bootstrap-worktree/<component>.sh` copies `.envrc` from the main checkout into the worktree and runs `direnv allow` there, so a delegate runs it first.

When you add an MCP server that uses a new `${VAR}`, add the variable to `.envrc.example` in the same change, and tell the team to update their `.envrc` and run `direnv allow`.

## Running the Admin app locally

The Admin app (`admin/`, ADR-0024) runs on port 3001 (`--strictPort`, so a busy port fails instead of moving), next to the user app on 3000, against the local API and the Clerk development instance.

1. In `backend/.env`, set `ADMIN_EMAILS` to your email and `ADMIN_CLIENT_URLS=http://localhost:3001` (see `backend/.env.example`). The API refuses to boot with `ADMIN_EMAILS` set and `ADMIN_CLIENT_URLS` empty, and the admin routes exist only when the four `R2_*` settings are set too (`docs/agents/deployment/images.md`).
2. Copy `admin/.env.example` to `admin/.env`. Copy the Clerk development publishable key across from `frontend/.env` (`scripts/clerk-setup.sh` writes only that one); the API address defaults to the local API.
3. Run `bun run dev` in `admin/`, open `http://localhost:3001`.

The Admin app has no sign-in page, so sign in at `http://localhost:3000` first with a listed email. Whether the development instance carries that session over to port 3001 has not been checked in a browser; if the Admin app shows not-found for a listed email, that is the first thing to suspect. A Guest and an unlisted user see the ordinary not-found page by design.
