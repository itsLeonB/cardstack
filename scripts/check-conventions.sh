#!/usr/bin/env bash
# Fails when a bullet in docs/agents/conventions/*.md is longer than MAX_BULLET characters.
# A bullet that long explains one module; move that to the module's header comment or an ADR (see general.md).
set -euo pipefail

MAX_BULLET=900
cd "$(dirname "$0")/.."

status=0
for file in docs/agents/conventions/*.md; do
  while IFS=: read -r line length; do
    echo "$file:$line: bullet is $length characters (max $MAX_BULLET); move the module explanation to its header comment or an ADR" >&2
    status=1
  done < <(LC_ALL=C.UTF-8 awk -v max="$MAX_BULLET" '/^- / && length($0) > max { print NR ":" length($0) }' "$file")
done
exit "$status"
