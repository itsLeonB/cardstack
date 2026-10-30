# Environment setup

Set up a fresh clone so agents and their MCP servers see the variables `.mcp.json` references. MCP servers read `${VAR}` placeholders from the environment of the process that starts them, so load the variables before you start your agent.

1. Install `direnv` and hook it into your shell (zsh: `eval "$(direnv hook zsh)"` in `~/.zshrc`). Restart the shell.
2. Install Serena (`serena`, `serena-hooks`) and `jq`. Run `scripts/lsp/install.sh` for the language servers; `--check` only reports what is missing.
3. Run `cp .envrc.example .envrc` and fill in your values. `.envrc` is gitignored; never commit it.
4. Run `direnv allow` in the repo root, and again after every `.envrc` change.
5. Start your agent from that shell, inside the repo.

Worktrees don't hold gitignored files. A delegate runs `scripts/bootstrap-worktree/<component>.sh` first: it copies `.envrc` and the component's env files from the main checkout, runs `direnv allow`, and starts the compose stack's tooling container (the frontend script also installs dependencies in it).

When an MCP server needs a new `${VAR}`, add it to `.envrc.example` in the same change and tell the team to update `.envrc` and run `direnv allow`.

## Run the Admin app locally

The Admin app (`admin/`, ADR-0024) runs on port 3001 with `--strictPort`, next to the user app on 3000. It uses the local API and the Clerk development instance.

1. In `backend/.env`, set `ADMIN_EMAILS` to your email and `ADMIN_CLIENT_URLS=http://localhost:3001` (see `backend/.env.example`). The API refuses to boot with `ADMIN_EMAILS` set and `ADMIN_CLIENT_URLS` empty. Admin routes also need the four `R2_*` settings (`docs/agents/deployment/images.md`).
2. Run `cp admin/.env.example admin/.env`. Copy the Clerk development publishable key from `frontend/.env`; `scripts/clerk-setup.sh` writes only that key. The API address defaults to the local API.
3. Run `bun run dev` in `admin/` and open `http://localhost:3001`.

The Admin app has no sign-in page: sign in at `http://localhost:3000` with a listed email first. Unverified in a browser: whether the development instance carries that session to port 3001. If a listed email gets not-found, suspect that first. Guests and unlisted users always get the ordinary not-found page.

## The compose stack

Docker with the Compose plugin is the only tool the checks need: the database, API, apps and tooling run in containers (`docs/agents/testing.md`). The host keeps its own language toolchains (Go, bun) because Serena's language servers run on the host and need them; nothing else does.

- **Env files.** The stack reads `backend/.env` when it exists, over the placeholders in `compose.defaults.env` that let the API boot without it. `frontend/.env` supplies the Clerk publishable key the end-to-end step bakes into the app. A missing file is fine; the guest checks still run.
- **Isolation.** Each worktree has its own stack, named after its directory, with no host port published.
- **Your own browser.** Copy `compose.override.example.yaml` to the gitignored `compose.override.yaml`. It publishes ports 5432, 8080, 3000 and 3001, adds the localhost origins to the API's allowed client addresses, and rebuilds the apps with `http://localhost:8080` as the API address. The apps bake that address in at build time, so changing it, or the Clerk key, needs `docker compose up -d --build`. There is no hot reload: for that, keep running `bun run dev` natively.
- **Cleanup.** `docker compose down` removes the containers and the database; the named caches `cardstack-go-cache` and `cardstack-bun-cache` stay, and `docker volume rm` removes them.
