#!/usr/bin/env bash
# Runs scripts/preview-reconcile.py against every case directory under cases/.
# A case holds listing.json (the environment's variables) and either
# expected.json (the exact output) or expected-error (text the failure names),
# and optionally frontend-host (default: the one for PR 42) and manifest.yml
# (default: the real manifest).
# A successful case is also applied to its listing and run again, which must
# leave nothing to do. The script never sees a secret source: its output holds
# @@SOURCE@@ placeholders, resolved here the way the workflow resolves them.
# Also checks that the manifest classifies every variable in backend/.env.example.
# Needs python3 with PyYAML, and jq.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
manifest="$root/backend/infra/.railway/preview-vars.yml"
reconcile() { python3 -I "$root/scripts/preview-reconcile.py" --manifest "$case_manifest" --pr 42 "$@"; }

# The values the non-secret override sources resolve to. The secret ones
# (NEON_PASSWORD, CLERK_DEV_SECRET_KEY) are deliberately not set here.
export NEON_HOST=pr-host.neon.test NEON_DATABASE_NAME=pr_db
export NEON_ROLE_NAME=pr_role CLERK_DEV_ISSUER=https://dev.clerk.test
secrets='{"NEON_PASSWORD":"pr-secret","CLERK_DEV_SECRET_KEY":"sk_test_dev"}'
# A plan on stdin to the variables it would write, secrets filled in.
resolve() {
  jq --argjson s "$secrets" \
    '. as $p | reduce ($p.secrets | to_entries[]) as $e ($p.variables; .[$e.key] = $s[$e.value])'
}

failed=0
err="$(mktemp)"
trap 'rm -f "$err"' EXIT
fail() { echo "FAIL $name: $*" >&2; failed=1; }

for dir in "$here"/cases/*/; do
  name="$(basename "$dir")"
  host=cardstack-pr-42.vercel.app
  if [ -f "$dir/frontend-host" ]; then host="$(cat "$dir/frontend-host")"; fi
  case_manifest="$manifest"
  if [ -f "$dir/manifest.yml" ]; then case_manifest="$dir/manifest.yml"; fi
  if actual="$(reconcile --frontend-host "$host" "$dir/listing.json" 2>"$err")"; then
    if [ ! -f "$dir/expected.json" ]; then fail "succeeded, expected an error"; continue; fi
    diff <(jq -S . <<<"$actual") <(jq -S . "$dir/expected.json") >&2 || fail "output differs"
    applied="$(mktemp)"
    resolve <<<"$actual" >"$applied"
    again="$(reconcile --frontend-host "$host" "$applied")"
    [ "$(resolve <<<"$again" | jq -cS .)" = "$(jq -cS . "$applied")" ] || fail "second run changes something: $again"
    [ "$(jq -c .removals <<<"$again")" = '[]' ] || fail "second run removes something: $again"
    rm -f "$applied"
  else
    if [ ! -f "$dir/expected-error" ]; then fail "failed: $(cat "$err")"; continue; fi
    grep -qF "$(cat "$dir/expected-error")" "$err" || fail "error lacks: $(cat "$dir/expected-error")"
  fi
done

# Every variable backend/.env.example sets must be classified, so a new API
# setting cannot reach a preview unclassified.
name=env-example
unclassified="$(comm -23 \
  <(sed -n 's/^\([A-Z][A-Z0-9_]*\)=.*/\1/p' "$root/backend/.env.example" | sort -u) \
  <(python3 -I -c 'import sys, yaml
m = yaml.safe_load(open(sys.argv[1]))
print("\n".join(sorted({n for g in m.values() for n in g})))' "$manifest"))"
[ -z "$unclassified" ] || fail "not in the manifest: $(tr '\n' ' ' <<<"$unclassified")"

[ "$failed" = 0 ] && echo "preview-reconcile: all cases pass"
exit "$failed"
