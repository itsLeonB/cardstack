#!/usr/bin/env bash
# Verifies the frontend component. Run it from anywhere; it stops at the first failing command.
set -euo pipefail

cd "$(dirname "$0")/../../frontend"

run() {
  echo "+ $*"
  "$@"
}

run bun run lint
run bun run check
run bun run typecheck
run bun run test
run bun run build
