#!/usr/bin/env bash
# Verifies the frontend components: the user app (frontend/) and the Admin app (admin/).
# Run it from anywhere; it stops at the first failing command.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"

run() {
  echo "+ $*"
  "$@"
}

for pkg in frontend admin; do
  cd "$root/$pkg"
  run bun run lint
  run bun run check
  run bun run typecheck
  run bun run test
  run bun run build
done
