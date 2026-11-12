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

## Comments

### Investigation on base `20d395fd`

The exact verification-code timeout reproduced in the wrong-password test. The cause remains unproven; do not close this issue or treat account lockout as the cause of that earlier timeout. No fix remains in the helper, and all temporary `[DEBUG-signin]` probes were removed.

| Run | Result |
| --- | --- |
| Ticket command, `e2e/sign-in.spec.ts:42`, before rebuilding | 1 passed; this line currently selects the wrong-password test. |
| Both sign-in tests before rebuilding | 2 passed in 22.3 seconds. |
| `scripts/verification/frontend.sh e2e` | 72 passed, 17 failed in 2.5 minutes. The sign-in failures in this run were a missing email form and a missing guest redirect, not the code-field timeout. |
| Both sign-in tests, `--repeat-each=3 --workers=1 --retries=0` | 5 passed, 1 failed in 1.8 minutes. The wrong-password test timed out at `codeField.fill("424242")`, waiting for “Enter verification code.” |
| Wrong-password test alone, `--repeat-each=5 --workers=1 --retries=0` | 2 passed, 3 failed in 1.5 minutes: one exact code-field timeout and two missing-email-form failures. This reproduces the symptom without the happy-path test, but is not a fast, deterministic loop. |

Repeated wrong-password attempts subsequently locked the shared Clerk user. Clerk displayed a 60-minute lock; the user confirmed a manual unlock. Stop repeating this negative test against the shared user. No auth tests ran after that unlock, and CI was not run for this investigation.

Before further auth runs, isolate the wrong-password test with a fresh disposable Clerk development user for each attempt, including retries, and delete only that fixture-owned user during teardown. Keep the shared user for successful sign-ins. Keep lockout protection enabled: Testing Tokens do not promise a lockout bypass, and `+clerk_test` addresses prevent mail delivery rather than failed-password lockout. See [official-source research and recovery limits](../research/clerk-test-user-lockout.md).

Next investigation: implement the disposable-user fixture, then resume diagnosis of the original timeout. A form-transition race remains a hypothesis, not a confirmed finding. The missing-form failures are also unresolved. No lockout-setting change or automatic unlock of the shared user is part of the proposed fix.
