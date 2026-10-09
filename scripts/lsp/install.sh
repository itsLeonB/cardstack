#!/usr/bin/env bash
# Runs every component's language-server script. Pass --check to install nothing and only report.
set -uo pipefail

cd "$(dirname "$0")"
status=0
for script in ./*.sh; do
  [ "$script" = "./install.sh" ] && continue
  echo "== ${script#./}"
  "$script" "$@" || status=1
done
exit "$status"
