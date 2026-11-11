#!/bin/sh
# Sets up a dev environment for this project
# Non-interactive; safe to re-run. Intended to be run manually by devs.
set -e

echo "Installing Go language server (gopls)..."
if ! command -v go >/dev/null 2>&1; then
	echo "error: 'go' is not on PATH; install the Go toolchain first" >&2
	exit 1
fi
# Installed straight into /usr/local/bin (not GOPATH/bin) so it's resolvable
# by every process on this box without relying on a shell rc file having
# been sourced first
GOBIN=/usr/local/bin GOFLAGS=-mod=mod go install golang.org/x/tools/gopls@latest

if ! command -v npm >/dev/null 2>&1; then
	echo "error: 'npm' is not on PATH; install Node.js first" >&2
	exit 1
fi

echo "Installing TypeScript language server..."
npm install --global --no-fund --no-audit typescript typescript-language-server

echo "Installing YAML language server..."
npm install --global --no-fund --no-audit yaml-language-server

echo "Installing yamllint..."
if ! command -v pip >/dev/null 2>&1; then
	echo "error: 'pip' is not on PATH; install Python first" >&2
	exit 1
fi
pip install --user yamllint

echo "Enabling the tracked pre-commit checks (docs/git.md)..."
git config core.hooksPath .githooks

echo "Environment setup complete!"
