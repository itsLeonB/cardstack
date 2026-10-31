#!/usr/bin/env bash
# Runs scripts/preview-reconcile.py against every case directory under cases/.
# A case holds listing.json (the environment's variables) and either
# expected.json (the exact output) or expected-error (text the failure names).
# A successful case is also applied to its listing and run again, which must
# leave nothing to do. Needs python3 with PyYAML, and jq.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
reconcile=(python3 -I "$root/scripts/preview-reconcile.py"
  --manifest "$root/backend/infra/.railway/preview-vars.yml" --pr 42)

# The values the override sources resolve to.
export NEON_HOST=pr-host.neon.test NEON_DATABASE_NAME=pr_db NEON_PASSWORD=pr-secret
export NEON_ROLE_NAME=pr_role CLERK_DEV_ISSUER=https://dev.clerk.test CLERK_DEV_SECRET_KEY=sk_test_dev

failed=0
err="$(mktemp)"
trap 'rm -f "$err"' EXIT
fail() { echo "FAIL $name: $*" >&2; failed=1; }

for dir in "$here"/cases/*/; do
  name="$(basename "$dir")"
  if actual="$("${reconcile[@]}" "$dir/listing.json" 2>"$err")"; then
    if [ ! -f "$dir/expected.json" ]; then fail "succeeded, expected an error"; continue; fi
    diff <(jq -S . <<<"$actual") <(jq -S . "$dir/expected.json") >&2 || fail "output differs"
    again="$(jq --argjson plan "$actual" \
      '. + $plan.sets | del(.[$plan.removals[]])' "$dir/listing.json" \
      | "${reconcile[@]}")"
    [ "$(jq -c . <<<"$again")" = '{"removals":[],"sets":{}}' ] || fail "second run not empty: $again"
  else
    if [ ! -f "$dir/expected-error" ]; then fail "failed: $(cat "$err")"; continue; fi
    grep -qF "$(cat "$dir/expected-error")" "$err" || fail "error lacks: $(cat "$dir/expected-error")"
  fi
done
[ "$failed" = 0 ] && echo "preview-reconcile: all cases pass"
exit "$failed"
