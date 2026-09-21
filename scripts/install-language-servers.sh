#!/bin/sh
# Installs the Go, TypeScript, and YAML language servers Serena MCP relies on.
# Non-interactive; safe to re-run. Intended to be run manually by devs,
# e.g. as a Claude Code cloud session setup command.
set -e

echo "Installing Go language server (gopls)..."
if ! command -v go >/dev/null 2>&1; then
	echo "error: 'go' is not on PATH; install the Go toolchain first" >&2
	exit 1
fi
GOFLAGS=-mod=mod go install golang.org/x/tools/gopls@latest

if ! command -v npm >/dev/null 2>&1; then
	echo "error: 'npm' is not on PATH; install Node.js first" >&2
	exit 1
fi

echo "Installing TypeScript language server..."
npm install --global --no-fund --no-audit typescript typescript-language-server

echo "Installing YAML language server..."
npm install --global --no-fund --no-audit yaml-language-server

echo "Language servers installed successfully!"

gobin=$(go env GOBIN)
[ -n "$gobin" ] || gobin="$(go env GOPATH)/bin"
case ":$PATH:" in
*":$gobin:"*) ;;
*) echo "note: $gobin (gopls) is not on PATH; add it to use gopls from the shell" ;;
esac
