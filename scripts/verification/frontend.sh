#!/usr/bin/env bash
# Verifies the frontend components, the user app (frontend/) and the Admin app
# (admin/), in the compose stack's tooling container.
# Usage: frontend.sh [--package frontend|admin] [install|lint|format|typecheck|test|build|e2e]
# With no step it runs every step in order and stops at the first failure; with
# no package, both. Every step but e2e needs no API, so it starts only the
# tooling container and runs against the bind-mounted source, with no rebuild.
# e2e seeds the database, rebuilds the user app's image if its source changed
# and runs the Playwright suite against it (docs/agents/testing.md).
set -euo pipefail

source "$(dirname "$0")/lib.sh"

run() { run_in frontend-tools "$@"; }

packages=(frontend admin)
if [ "${1:-}" = --package ]; then
  packages=("${2:?--package needs frontend or admin}")
  shift 2
fi

install() {
  up frontend-tools
  RUN_DIR=$pkg run bun install --frozen-lockfile
}

step_lint() { RUN_DIR=$pkg run bun run lint; }
step_format() { RUN_DIR=$pkg run bun run check; }
step_typecheck() { RUN_DIR=$pkg run bun run typecheck; }
step_test() { RUN_DIR=$pkg run bun run test; }
step_build() {
  RUN_DIR=$pkg RUN_ENV="VITE_API_BASE_URL VITE_CLERK_PUBLISHABLE_KEY" \
    run bun run build
}

step_e2e() {
  # The user app's key, so the built app and the suite agree on the Clerk
  # instance. CI sets it; a local run reads it from the app's env file.
  if [ -z "${VITE_CLERK_PUBLISHABLE_KEY:-}" ] && [ -f "$root/frontend/.env" ]; then
    VITE_CLERK_PUBLISHABLE_KEY="$(sed -n 's/^VITE_CLERK_PUBLISHABLE_KEY=//p' "$root/frontend/.env")"
    export VITE_CLERK_PUBLISHABLE_KEY
  fi
  up api-ready web frontend-tools
  compose run --rm seed
  RUN_DIR=frontend \
    RUN_ENV="VITE_CLERK_PUBLISHABLE_KEY CLERK_SECRET_KEY E2E_CLERK_USER_EMAIL E2E_CLERK_USER_PASSWORD E2E_REQUIRE_SIGN_IN CI" \
    run env PLAYWRIGHT_BASE_URL=http://web:3000 PLAYWRIGHT_HTML_OPEN=never \
      bunx playwright test
}

steps=(lint format typecheck test build)
step=${1:-all}
case "$step" in
  all | e2e | install | lint | format | typecheck | test | build) ;;
  *)
    echo "usage: $0 [--package frontend|admin] [${steps[*]// /|}|e2e]" >&2
    exit 2
    ;;
esac

for pkg in "${packages[@]}"; do
  # admin/ is absent in worktrees cut before the Admin app existed.
  [ -d "$root/$pkg" ] || continue
  [ "$step" = e2e ] && continue
  install
  if [ "$step" = install ]; then
    continue
  elif [ "$step" = all ]; then
    for s in "${steps[@]}"; do "step_$s"; done
  else
    "step_$step"
  fi
done

if [ "$step" = e2e ] || { [ "$step" = all ] && [[ " ${packages[*]} " == *" frontend "* ]]; }; then
  step_e2e
fi
