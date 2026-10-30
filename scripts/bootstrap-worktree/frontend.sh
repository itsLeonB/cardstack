#!/usr/bin/env bash
# Prepares a fresh frontend worktree (frontend/ and the Admin app's admin/): a new worktree holds
# only commits, so copy the gitignored env files from the main checkout, start the compose
# stack's tooling container and install dependencies in it (docs/agents/testing.md).
# Idempotent.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
# The first entry of `git worktree list` is always the main checkout.
main=$(git worktree list --porcelain | awk '/^worktree /{print substr($0, 10); exit}')

for f in .envrc frontend/.env admin/.env; do
  if [ ! -e "$root/$f" ] && [ -e "$main/$f" ]; then
    cp "$main/$f" "$root/$f"
    echo "copied $f from the main checkout"
  fi
done

# `.envrc` is trusted per directory, so a copied one must be allowed again here.
if [ -e "$root/.envrc" ] && command -v direnv >/dev/null 2>&1; then
  direnv allow "$root"
fi

cd "$root"
source scripts/verification/lib.sh
# Starts the tooling container and installs both packages' dependencies in it.
scripts/verification/frontend.sh install
