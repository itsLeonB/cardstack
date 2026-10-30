#!/usr/bin/env bash
# Verifies the backend component in the compose stack's tooling container.
# Usage: backend.sh [build|vet|format|lint|test]
# With no argument it runs every step in order and stops at the first failure.
# A step that needs no database starts only the tooling container.
set -euo pipefail

source "$(dirname "$0")/lib.sh"

run() { run_in backend-tools "$@"; }

step_build() { up backend-tools; run go build ./...; }
step_vet() { up backend-tools; run go vet ./...; }
step_format() {
  up backend-tools
  run sh -c 'out=$(gofmt -l .); [ -z "$out" ] || { echo "gofmt: $out"; exit 1; }'
}
step_lint() { up backend-tools; run golangci-lint run ./... --timeout=5m; }
step_test() { up db backend-tools; run go test -race ./...; }

steps=(build vet format lint test)
case "${1:-all}" in
  all) for s in "${steps[@]}"; do "step_$s"; done ;;
  build | vet | format | lint | test) "step_$1" ;;
  *)
    echo "usage: $0 [${steps[*]// /|}]" >&2
    exit 2
    ;;
esac
