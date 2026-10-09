#!/usr/bin/env bash
# Verifies the backend component. Run it from anywhere; it stops at the first failing command.
set -euo pipefail

cd "$(dirname "$0")/../../backend"

run() {
  echo "+ $*"
  "$@"
}

run go build ./...
run go vet ./...
run test -z "$(gofmt -l .)"
run golangci-lint run ./... --timeout=5m
run go test -race ./...
