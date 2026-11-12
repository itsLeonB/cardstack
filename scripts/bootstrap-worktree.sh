#!/usr/bin/env bash
# Prepares a fresh worktree for any component: a new worktree holds only
# commits, so copy the gitignored env files from the main checkout, then start
# the tooling containers and install frontend dependencies (docs/agents/
# testing.md). Never overwrites an existing env file and never prints values.
# Idempotent.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
# The first entry of `git worktree list` is always the main checkout.
main=$(git worktree list --porcelain | awk '/^worktree /{print substr($0, 10); exit}')

if [ "$root" = "$main" ]; then
  echo "bootstrap-worktree: already in the main checkout, nothing to prepare" >&2
  exit 0
fi

for f in .envrc backend/.env frontend/.env admin/.env; do
  if [ -e "$root/$f" ]; then
    echo "kept $f (already exists)"
  elif [ -e "$main/$f" ]; then
    cp "$main/$f" "$root/$f"
    echo "copied $f from the main checkout"
  else
    echo "skipped $f (not in the main checkout)"
  fi
done

# `.envrc` is trusted per directory, so a copied one must be allowed again here.
if [ -e "$root/.envrc" ] && command -v direnv >/dev/null 2>&1; then
  direnv allow "$root"
fi

cd "$root"
source scripts/verification/lib.sh
up backend-tools frontend-tools
# Installs both frontend packages' dependencies in the tooling container.
scripts/verification/frontend.sh install
