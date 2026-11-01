#!/usr/bin/env bash
# Self-check for .pi/extensions/deny-big-output.ts (needs node >= 22.6 for --experimental-strip-types).
set -eu
root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
dir=$(mktemp -d); trap 'rm -rf "$dir"' EXIT
git -C "$dir" init -q
echo hi >"$dir/small.txt"; echo x >"$dir/big.txt"
git -C "$dir" add -A
git -C "$dir" -c user.email=a@b -c user.name=t commit -qm init
head -c 50000 /dev/zero | tr '\0' y >"$dir/big.txt"

node --experimental-strip-types --no-warnings - "$root/.pi/extensions/deny-big-output.ts" "$dir" <<'EOF'
const [file, dir] = process.argv.slice(2);
// The extension only imports a type from pi, which strip-types erases.
const { decide } = await import(file);
const bash = (command) => ["bash", { command }];
const cases = [
  ["cat big", bash("cat big.txt"), true],
  ["cat small", bash("cat small.txt"), false],
  ["cat big piped", bash("cat big.txt | head -5"), false],
  ["cat big BIG_OK", bash("BIG_OK=1 cat big.txt"), false],
  ["git diff big", bash("git diff"), true],
  ["git diff --stat", bash("git diff --stat"), false],
  ["git diff small path", bash("git diff -- small.txt"), false],
  ["read big no limit", ["read", { path: "big.txt" }], true],
  ["read big limit", ["read", { path: "big.txt", limit: 100 }], false],
  ["read small", ["read", { path: "small.txt" }], false],
  ["other tool", ["edit", { path: "x" }], false],
];
let fail = 0;
for (const [name, [tool, input], deny] of cases) {
  if ((decide(tool, input, dir) !== "") !== deny) { console.log(`FAIL ${name}: deny should be ${deny}`); fail = 1; }
}
if (!fail) console.log("ok");
process.exit(fail);
EOF
