# Environment setup

Set up a fresh clone so agents and their MCP servers see the variables `.mcp.json` references. MCP servers read `${VAR}` placeholders from the environment of the process that starts them, so load the variables before you start your agent.

1. Install `direnv` and hook it into your shell (zsh: `eval "$(direnv hook zsh)"` in `~/.zshrc`). Restart the shell.
2. Install Serena (`serena`, `serena-hooks`) and `jq`. Run `scripts/lsp/install.sh` for the language servers; `--check` only reports what is missing.
3. Run `cp .envrc.example .envrc` and fill in your values. `.envrc` is gitignored; never commit it.
4. Run `direnv allow` in the repo root, and again after every `.envrc` change.
5. Start your agent from that shell, inside the repo.

Worktrees don't hold gitignored files. A delegate runs `scripts/bootstrap-worktree/<component>.sh` first: it copies `.envrc` from the main checkout and runs `direnv allow`.

When an MCP server needs a new `${VAR}`, add it to `.envrc.example` in the same change and tell the team to update `.envrc` and run `direnv allow`.

## Run the Admin app locally

The Admin app (`admin/`, ADR-0024) runs on port 3001 with `--strictPort`, next to the user app on 3000. It uses the local API and the Clerk development instance.

1. In `backend/.env`, set `ADMIN_EMAILS` to your email and `ADMIN_CLIENT_URLS=http://localhost:3001` (see `backend/.env.example`). The API refuses to boot with `ADMIN_EMAILS` set and `ADMIN_CLIENT_URLS` empty. Admin routes also need the four `R2_*` settings (`docs/agents/deployment/images.md`).
2. Run `cp admin/.env.example admin/.env`. Copy the Clerk development publishable key from `frontend/.env`; `scripts/clerk-setup.sh` writes only that key. The API address defaults to the local API.
3. Run `bun run dev` in `admin/` and open `http://localhost:3001`.

The Admin app has no sign-in page: sign in at `http://localhost:3000` with a listed email first. Unverified in a browser: whether the development instance carries that session to port 3001. If a listed email gets not-found, suspect that first. Guests and unlisted users always get the ordinary not-found page.
