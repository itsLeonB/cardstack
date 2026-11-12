# 02: Make the local sign-in browser test reliable

**What to build:** A developer can run the real Clerk sign-in browser test locally and get a reliable result that matches the supported test setup.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

## Reproduction

1. Set up the compose stack and real Clerk test credentials as described in `docs/agents/testing.md`. Do not print credentials or include them in test artifacts.
2. Run `scripts/verification/frontend.sh e2e` once to build and seed the stack.
3. From the repository root, run `docker compose exec -w /workspace/frontend -e PLAYWRIGHT_BASE_URL=http://web:3000 frontend-tools bunx playwright test e2e/sign-in.spec.ts:42 --project=chromium-signed-in --retries=0`.
4. Observed locally: the test times out after 30 seconds waiting for “Enter verification code” in the Clerk sign-in flow. The same test failed in isolation as well as in the full suite.

## Evidence

- The API stayed running with no restarts or matching fatal signatures; the observed failure does not match the known API crash and restart issues.
- The [E2E CI run on base commit `a3eb876c`](https://github.com/itsLeonB/cardstack/actions/runs/43177917909) passed all 89 tests. The sign-in spec and helper were unchanged in the subsequent Trade fix. This does not prove the cause or establish that the current commit passes CI.
- The cause is unknown. A local Clerk or test-flow limitation is possible but not confirmed.

## Acceptance criteria

- [ ] Identify the cause of the timeout or document the external condition that prevents a reliable local run.
- [ ] Make the sign-in test pass reliably in the supported local setup without weakening its real Clerk coverage, or document a concrete, safe workaround if the cause is outside this repo.
- [ ] Run the focused sign-in test and the full frontend E2E suite; record results and any remaining CI or local limits.
