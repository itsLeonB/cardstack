# 12 Guard extension hardening: test seams, push guard, CI

Type: task
Status: open

Should the two `.pi/extensions/` guards share one importable rule module and run their self-checks in CI?

## Problem

- `deny-big-output.ts` exports `decide` only so `scripts/tests/deny-big-output/run.sh` can import it: a test-only seam in a production file.
- `scripts/tests/block-commit-hooks/run.sh` scrapes `COMMIT_OR_MERGE` out of the `.ts` source with a regex and `eval`s it, so a reformat breaks the check silently. It covers the regex only, not branch detection.
- Neither script runs in CI. `preview-reconcile` is the only self-check wired (`.github/workflows/preview-vars-ci.yml`), and nothing enforces node >= 22.6 for `--experimental-strip-types`.
- `block-commit-on-main.ts` blocks `commit` and `merge` on `main` only. The shell hook it replaced also blocked `git push` on `main` and followed `git -C <path>` to the target checkout's branch. The extension reads `ctx.cwd` only, so `git -C <other-checkout> commit` passes the guard.

## Draft approach

1. Move the pure logic (`decide`, the commit/merge/push classifier) into `.pi/extensions/guard-rules.ts`, imported by both extensions and by the self-checks. No test-only export remains.
2. Cover push: refuse `git push` while the session is on `main`, and resolve `-C <path>` to that checkout's branch before deciding.
3. Add `.github/workflows/pi-extensions-ci.yml` shaped like `preview-vars-ci.yml`: `paths` filtered to `.pi/extensions/**`, `scripts/tests/**` and the workflow itself; one `run.sh` per extension; node pinned to a version that supports `--experimental-strip-types`.
4. Name both scripts in `docs/agents/testing.md`, so a reader can find them.

## Open

`merge-ticket` is a Go CLI (`tools/cmd/merge-ticket`). Keep it Go and duplicate the protected-branch constant, or port its checks to TS so `guard-rules.ts` is the single source? A Pi extension cannot replace it: extensions hook the agent runtime, and humans and non-Pi agents run `merge-ticket` directly.
