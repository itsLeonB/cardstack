#!/bin/sh
# Sets up a dev environment for this project: language servers, RTK, Serena,
# ADHD mode, and a set of shared Claude Code / Codex skills.
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

echo "Installing yamllint..."
if ! command -v pip >/dev/null 2>&1; then
	echo "error: 'pip' is not on PATH; install Python first" >&2
	exit 1
fi
pip install --user yamllint

echo "Installing RTK..."
curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh | sh

echo "Installing Serena..."
if ! command -v uv >/dev/null 2>&1; then
	echo "error: 'uv' is not on PATH; install uv first" >&2
	exit 1
fi
uv tool install -p 3.13 serena-agent
serena init

echo "Enabling ADHD always-on mode..."
mkdir -p ~/.claude && touch ~/.claude/.i-have-adhd-always

echo "Installing shared skills..."
npx skills add vercel-labs/skills --skill "find-skills" -g -a claude-code -a codex -y
npx skills add dmmulroy/anti-slop --skill "install-anti-slop" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "grill-me" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "grilling" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "handoff" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "teach" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "to-questionnaire" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "wait-what" -g -a claude-code -a codex -y
npx skills add mattpocock/skills --skill "writing-for-agents" -g -a claude-code -a codex -y
npx skills add anthropics/skills --skill "skill-creator" -g -a claude-code -a codex -y
npx skills add anthropics/claude-plugins-official --skill "claude-automation-recommender" -g -a claude-code -a codex -y

echo "Environment setup complete!"

gobin=$(go env GOBIN)
[ -n "$gobin" ] || gobin="$(go env GOPATH)/bin"
case ":$PATH:" in
*":$gobin:"*) ;;
*) echo "note: $gobin (gopls) is not on PATH; add it to use gopls from the shell" ;;
esac
