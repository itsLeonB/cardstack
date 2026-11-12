# 02: Make the local sign-in browser test reliable

**What to build:** A developer can run the real Clerk sign-in browser test locally and get a reliable result that matches the supported test setup.

**Blocked by:** None (can start immediately).

**Status:** needs-info

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

### Disposable-user isolation and follow-up

The wrong-password test now creates a fresh Clerk development user per attempt and deletes its returned ID in teardown. Successful sign-ins keep the shared user. An interrupted process or a lost creation response can leave a disposable user behind; no automatic recovery is included.

The original code-field timeout reproduced with a disposable user: the failure snapshot showed the password step after the helper selected the code field. This confirms a form-transition race in that run, not that the earlier timeout had the same cause. The helper now checks the active prompt again and waits for a successful code-send response before entering a code.

| Run | Result |
| --- | --- |
| Both sign-in tests, after isolation | 2 passed. |
| Wrong-password test repeated three times before the prompt fix | 2 passed, 1 code-field timeout. |
| Wrong-password test repeated five times after the prompt fix | 5 passed. |
| Full frontend verification before the final prompt fix | 78 passed, 11 failed, including a happy-path code-send error and ten guest failures. |
| `scripts/verification/frontend.sh --package frontend` after the fix | All checks passed; E2E 89 passed. |

CI was not run. These runs do not prove the intermittent failure is eliminated.

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

### Recheck on base `97cee565`

This base already includes the disposable-user fixture, prompt recheck, code-send wait and extra fixture setup time from `cf0f3a5d` and `ff3fefc1`. No code changed during this recheck. The timeout did not reproduce; passing runs do not prove that these fixes removed every intermittent failure.

All focused runs used the compose stack, real Clerk, `E2E_REQUIRE_SIGN_IN=true` and no retries. Failed-password attempts used only fresh disposable users; the shared user received only successful-password attempts. No cleanup failure was reported. Console output was redacted before display; focused runs used the line reporter and the untraced signed-in project.

| Command | Result |
| --- | --- |
| `scripts/verification/frontend.sh e2e` | 89 passed; 106 seconds including stack preparation, 1.3 minutes in Playwright. |
| `docker compose exec -T -w /workspace/frontend -e PLAYWRIGHT_BASE_URL=http://web:3000 -e E2E_REQUIRE_SIGN_IN=true frontend-tools bunx playwright test e2e/sign-in.spec.ts --project=chromium-signed-in --retries=0 --repeat-each=5 --workers=1 --reporter=line` | 10 passed in 68.8 seconds, five runs of each sign-in test. |
| `docker compose exec -T -w /workspace/frontend -e PLAYWRIGHT_BASE_URL=http://web:3000 -e E2E_REQUIRE_SIGN_IN=true frontend-tools bunx playwright test e2e/sign-in.spec.ts --grep 'a wrong password' --project=chromium-signed-in --retries=0 --repeat-each=20 --workers=2 --reporter=line` | 20 passed in 87.7 seconds. |

Use the spec path or test name, not the old `:42` selector: fixture setup now occupies that line.

Diagnosis stopped at Phase 1: the existing tests exercise the reported flow, but there is no reproduced failure or short, repeatable failing command on this base. Phases 2–5 cannot proceed without that signal. No new cause is established, no speculative fix or new regression test was added, and no debug instrumentation remains. CI was not run.

To resume, provide a fresh failure on this base or newer: commit, exact command and redacted error with the active Clerk prompt. Do not share a raw HAR, trace, HTML report, cookies, headers, email or password. Keep this issue open until a failing run supplies the missing evidence.
