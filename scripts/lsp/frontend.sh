#!/usr/bin/env bash
# Installs the language servers Serena needs for the frontend component (typescript).
# Idempotent and non-interactive. `--check` installs nothing and exits non-zero while a server is missing.
set -euo pipefail

LANGUAGES="typescript"

check_typescript() { command -v typescript-language-server >/dev/null 2>&1; }

install_typescript() {
  command -v npm >/dev/null 2>&1 || { echo "error: 'npm' is not on PATH; install Node.js first" >&2; exit 1; }
  npm install --global --no-fund --no-audit typescript typescript-language-server
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
