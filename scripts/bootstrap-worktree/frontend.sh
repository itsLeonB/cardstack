#!/usr/bin/env bash
# Prepares a fresh frontend worktree: a new worktree holds only commits, so install dependencies
# and copy the gitignored env files from the main checkout. Idempotent.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
# The first entry of `git worktree list` is always the main checkout.
main=$(git worktree list --porcelain | awk '/^worktree /{print substr($0, 10); exit}')

for f in .envrc frontend/.env; do
  if [ ! -e "$root/$f" ] && [ -e "$main/$f" ]; then
    cp "$main/$f" "$root/$f"
    echo "copied $f from the main checkout"
  fi
done

# `.envrc` is trusted per directory, so a copied one must be allowed again here.
if [ -e "$root/.envrc" ] && command -v direnv >/dev/null 2>&1; then
  direnv allow "$root"
fi

cd "$root/frontend"
bun install --frozen-lockfile
