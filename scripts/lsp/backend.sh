#!/usr/bin/env bash
# Installs the language servers Serena needs for the backend component (go yaml).
# Idempotent and non-interactive. `--check` installs nothing and exits non-zero while a server is missing.
set -euo pipefail

LANGUAGES="go yaml"

check_go() { command -v gopls >/dev/null 2>&1; }

install_go() {
  command -v go >/dev/null 2>&1 || { echo "error: 'go' is not on PATH; install the Go toolchain first" >&2; exit 1; }
  # /usr/local/bin is on every process's PATH. Serena probes each server once at session start and a
  # failed probe is not recoverable in-session, so gopls must resolve without a shell rc file.
  GOBIN=/usr/local/bin GOFLAGS=-mod=mod go install golang.org/x/tools/gopls@latest
}

check_yaml() { command -v yaml-language-server >/dev/null 2>&1; }

install_yaml() {
  command -v npm >/dev/null 2>&1 || { echo "error: 'npm' is not on PATH; install Node.js first" >&2; exit 1; }
  npm install --global --no-fund --no-audit yaml-language-server
}

missing=0
for lang in $LANGUAGES; do
  if "check_$lang"; then
    echo "ok: $lang"
  elif [ "${1:-}" = "--check" ]; then
    echo "missing: $lang" >&2
    missing=1
  else
    "install_$lang"
    "check_$lang" || { echo "error: $lang server still missing after install" >&2; exit 1; }
  fi
done
exit "$missing"
