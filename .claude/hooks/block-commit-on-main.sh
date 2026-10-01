#!/usr/bin/env bash
# PreToolUse(Bash): refuse `git commit`/`git merge` while the session's branch is main.
p=$(cat)
jq -r '.tool_input.command // ""' <<<"$p" | grep -Eq 'git( +-C +[^ ]+)? +(commit|merge)\b' || exit 0
d=$(jq -r '.cwd // "."' <<<"$p")
if [ "$(git -C "$d" branch --show-current 2>/dev/null)" = main ]; then
  echo "Blocked: never commit or merge on main. Create a feature branch first: git switch -c <type>/<name> (docs/agents/orchestration.md)." >&2
  exit 2
fi
exit 0
