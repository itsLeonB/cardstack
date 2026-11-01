#!/usr/bin/env bash
# Self-check for .pi/extensions/deny-shell-code-read.ts (needs node >= 22.6 for --experimental-strip-types).
set -eu
root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)

node --experimental-strip-types --no-warnings - "$root/.pi/extensions/deny-shell-code-read.ts" <<'EOF'
const [file] = process.argv.slice(2);
// The extension only imports a type from pi, which strip-types erases.
const { decide } = await import(file);
const bash = (command) => decide("bash", { command }) !== "";
const cases = [
  ["sed -n code", "sed -n 1,40p backend/internal/app.go", true],
  ["cat code", "cat frontend/src/app.tsx", true],
  ["head code", "head -n 20 cardstack.ts", true],
  ["tail code", "tail -5 lib.js", true],
  ["nl code", "nl -ba page.tsx", true],
  ["less code", "less app.go", true],
  ["cd then sed", "cd backend && sed -n 1,40p internal/app.go", true],
  ["wrapped sed", "timeout 60 sed -n 1,5p src/app.ts", true],
  ["rtk cat", "rtk cat src/app.ts", true],
  ["piped cat", "cat src/app.ts | head -5", true],
  ["quoted path", "cat 'src/app.ts'", true],
  ["sed -i code", "sed -i 's/a/b/' internal/app.go", true],
  ["sed -i glob flag", "sed -Ei 's/a/b/g' internal/app.go", true],
  ["escaped", "SERENA_OK=1 cat src/app.ts", false],
  ["escaped mid-chain", "cd backend && SERENA_OK=1 sed -n 1,9p app.go", false],
  ["cat non-code", "cat package.json", false],
  ["cat markdown", "cat docs/agents/testing.md", false],
  ["cat yaml", "cat .github/workflows/backend-ci.yml", false],
  ["heredoc write", "cat > internal/new.go <<'EOF'", false],
  ["append write", "cat >> notes.md <<'EOF'", false],
  ["sed non-code", "sed -n 1,20p compose.yaml", false],
  ["sed in-place non-code", "sed -i 's/x/y/' compose.yaml", false],
  ["git status", "git status --short", false],
  ["grep code", "grep -n Foo backend/app.go", false],
  ["go test", "cd backend && go test ./...", false],
  ["other tool", "", false],
];
let fail = 0;
for (const [name, command, deny] of cases) {
  const blocked = bash(command);
  if (blocked !== deny) {
    console.log(`FAIL ${name}: expected block=${deny}, got ${blocked} (${command || "<other tool>"})`);
    fail = 1;
  }
}
if (decide("read", { path: "src/app.ts" }) !== "") {
  console.log("FAIL read tool must not be guarded here: this extension covers bash only");
  fail = 1;
}
if (!fail) console.log("ok");
process.exit(fail);
EOF
