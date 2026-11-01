#!/usr/bin/env bash
# Self-check for the regex in .pi/extensions/block-commit-on-main.ts.
set -u
root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
fail=0

# expect <blocked|allowed> <command>
expect() {
  local got=allowed
  node -e '
    const m = require("fs").readFileSync(process.argv[1], "utf8").match(/COMMIT_OR_MERGE = (\/.*\/);/);
    process.exit(eval(m[1]).test(process.argv[2]) ? 0 : 1);
  ' "$root/.pi/extensions/block-commit-on-main.ts" "$2" && got=blocked
  [ "$got" = "$1" ] || { echo "FAIL (expected $1): $2"; fail=1; }
}

for c in "git merge-base --is-ancestor a b" "git -C /tmp/x merge-base a b" "git commit-graph write"; do
  expect allowed "$c"
done
for c in "git commit -m x" "git commit" "git -C /tmp/x merge feat"; do
  expect blocked "$c"
done

[ $fail = 0 ] && echo "ok"
exit $fail
