#!/usr/bin/env bash
# PreToolUse(Bash): refuse `git commit`, `git merge` and `git push` while the session's branch is protected,
# and refuse a push that names a protected branch.
PROTECTED="main"

p=$(cat)
cmd=$(jq -r '.tool_input.command // ""' <<<"$p")
# Match `git` only where a command starts (line start, or after ; & | ( $( and VAR=x prefixes), so text inside
# an echo or a heredoc that merely mentions `git commit` does not trigger the hook.
git_re='(^|[;&|(]|\$\()[[:space:]]*([A-Za-z_][A-Za-z0-9_]*=[^ ]* +)*git( +-C +[^ ]+)?'
grep -Eq "$git_re +(commit|merge|push)\b" <<<"$cmd" || exit 0

d=$(jq -r '.cwd // "."' <<<"$p")
# `git -C <path> ...` acts on <path>, not on the session's directory.
c=$(grep -oE 'git +-C +[^ ]+' <<<"$cmd" | head -1 | awk '{print $3}')
[ -n "$c" ] && d=$(git -C "$d" -C "$c" rev-parse --show-toplevel 2>/dev/null || echo "$d")
branch=$(git -C "$d" branch --show-current 2>/dev/null)

protected() { for b in $PROTECTED; do [ "$1" = "$b" ] && return 0; done; return 1; }
block() {
  echo "Blocked: $1. Create a feature branch first: git switch -c <type>/<name> (docs/agents/orchestration.md)." >&2
  exit 2
}

if grep -Eq "$git_re +(commit|merge)\b" <<<"$cmd"; then
  # A command that first creates a branch (`git switch -c x && ... git commit`) commits off the protected branch.
  grep -Eq "$git_re +(switch +(-c|-C|--create)|checkout +(-b|-B))\b" <<<"$cmd" && exit 0
  protected "$branch" && block "never commit or merge on protected branch '$branch'"
fi

if grep -Eq "$git_re +push\b" <<<"$cmd"; then
  protected "$branch" && block "never push from protected branch '$branch'"
  for b in $PROTECTED; do
    e=${b//./\\.}
    grep -Eq "push\b[^;&|]*[ :/]$e([ ;&|]|\$)" <<<"$cmd" && block "never push to protected branch '$b'"
  done
fi
exit 0
