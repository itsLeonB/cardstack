# Applying the security-guidance plugin without the per-turn Stop review

Research note, 2026-10-05. Plugin: `security-guidance@claude-plugins-official` v2.0.9. In this note `<plugin>` means `~/.claude/plugins/cache/claude-plugins-official/security-guidance/2.0.9`. A claim without a file:line or URL is marked "unverified".

## Summary and recommendation

**The 63 review sessions in the transcripts are not the Stop review.** Their prompt "Review this change for security vulnerabilities." is the agentic commit/push reviewer (`<plugin>/hooks/llm.py:1381` and `review_api.py:167`, called only from `security_reminder_hook.py:1395` commit review and `:1796` push sweep). The Stop review is a different prompt ("You are a security expert reviewing ...", `llm.py:1035`) sent as a plain `POST /v1/messages` (`llm.py:652`) on first-party auth, so it leaves no `sdk-py` transcript. Every one of the 63 sessions starts within seconds of a `git commit`/`git push` Bash call in the main transcripts (e.g. commit 2026-09-18T16:24:02Z, review session 16:24:05Z).

The plugin's own log `~/.claude/security/log.txt` (+`.1`, covering 2026-09-18 onward) shows both streams. Stop hook: 231 LLM reviews (`Stop hook: LLM reviews took`), 225 "no security issues found", 2 API failures, and the 5 "LLM code review found" lines all fall between 2026-09-18 and 2026-09-21. Commit review: 115 "detected git commit in command" hits, roughly 63 of which ran an agentic Opus session.

**Recommendation.** Keep the plugin, turn off only the Stop review with the plugin's own switch, and add a path-filtered PR-time CI backstop:

1. Add `"env": {"ENABLE_STOP_REVIEW": "0"}` to `.claude/settings.json`. This removes the ~231 per-turn single-shot calls and keeps the free pattern reminders (including the `.github/workflows/` reminder) and the commit/push agentic review that produced all three real findings.
2. Optionally add `.github/workflows/security-review.yml` (drafted below, passes `yamllint -c .yamllint.yaml`) so edits to workflows, scripts, auth and config are reviewed once per push on the PR, including commits Claude did not make through its Bash tool (those are not reviewed by the plugin, `https://code.claude.com/docs/en/security-guidance` "On each commit or push Claude makes").
3. Do not expect the plugin to reduce commit-review volume: it has no path filter, no diff-hash dedupe on that path and no debounce. Its only knobs there are the per-session hourly cap, the model and the on/off switches (table below).

Per-hook disabling of a plugin's hook from project settings is not possible (section 2c), so the env var is the only per-layer mechanism.

## 1. What the plugin supports

### 1a. Environment variables (all read at hook start; no settings file of its own)

The README states "All configuration is via environment variables" (`<plugin>/README.md:27`). The marketplace copy `~/.claude/plugins/marketplaces/claude-plugins-official/plugins/security-guidance/` is byte-identical to 2.0.9 (`diff -rq` shows only the cache's `.in_use` marker), and `marketplace.json` lists version 2.0.9, so there is no drift to report.

| Variable | Default | Effect | Source |
|---|---|---|---|
| `ENABLE_STOP_REVIEW` | on (`!= "0"`) | Skips only the Stop/SubagentStop LLM review (`_skip(50)`), after the state snapshot, so no LLM call is made | `hooks/security_reminder_hook.py:169`, `:1981-1984`; README:58 |
| `ENABLE_CODE_SECURITY_REVIEW` | on | Master switch for all LLM reviews, Stop and commit/push | `security_reminder_hook.py:145-146`, `:1973-1975`; README:57 |
| `ENABLE_COMMIT_REVIEW` | on | Disables commit review and also the push sweep | `security_reminder_hook.py:161`, `:826-827`, `:942-950`; README:59 |
| `ENABLE_PATTERN_RULES` | on | Disables the per-edit regex reminders | `security_reminder_hook.py:150-151`; README:56 |
| `SECURITY_GUIDANCE_DISABLE` / `ENABLE_SECURITY_REMINDER=0` | unset | Whole-plugin kill switch | `security_reminder_hook.py:176-180` |
| `SG_AGENTIC_COMMIT_REVIEW` | on | `0`/`off` makes commit and push reviews use the single-shot HTTP reviewer instead of an Agent SDK session (not "off") | `llm.py:1231-1241`; used at `security_reminder_hook.py:1393`, `:1795` |
| `SG_PUSH_SWEEP` | on | `0`/`off` disables only the push-sweep review | `security_reminder_hook.py:818-830` |
| `MAX_COMMIT_REVIEWS_PER_HOUR` (legacy `MAX_COMMIT_REVIEWS_PER_SESSION`) | 20 | Rolling cap per session and window; over cap the review is skipped | `security_reminder_hook.py:713-716`, `:344-352`, `:1369-1374` |
| `COMMIT_REVIEW_RATE_WINDOW_S` | 3600 | Window for the cap above | `security_reminder_hook.py:717-719` |
| `SECURITY_REVIEW_MODEL` | newest Opus; fallback `claude-opus-5-5` | Model for the single-shot (Stop) review; `~/.claude/security/review_model.json` currently caches `claude-opus-5-5` | `llm.py:130-143`; README:29-49 |
| `SG_AGENTIC_MODEL` | `opus` alias | Model for the agentic commit/push reviewer | `llm.py:1359-1360` |
| `SG_AGENTIC_MAX_TURNS` | 18 | Tool-use turns per agentic review | `llm.py:1361` |
| `SG_AGENTIC_NO_RACE` / `SG_AGENTIC_RACE_DELAY_S` | race on / 180 | After 180 s a single-shot review races the agentic one, so a slow review can cost two calls | `security_reminder_hook.py:985-988` |
| `SG_DUAL_OR` | off | Two parallel reviews, about 2x cost | `llm.py:765-767`; README:61-67 |
| `MAX_STOP_HOOK_FIRINGS` | 3 | Max Stop re-fires per loop (counter expires after `STOP_LOOP_STATE_TTL_SEC` = 120 s) | `security_reminder_hook.py:185`, `:1968-1970`; `hooks/diffstate.py:29` |
| `MAX_DIFF_FILES` | 30 | Files sent per Stop review; more files are prioritised by risky path tokens, over 300 are skipped | `security_reminder_hook.py:190`, `:2054-2070`; `hooks/gitutil.py:579-640` |

### 1b. Path filters, throttling, dedupe, on-demand mode

- **Path filters for the LLM reviews: none configurable.** Only a hard-coded skip list applies (`node_modules/ dist/ build/ .next/ vendor/ __generated__/ target/` etc., and `.min.js .d.ts .lock .pb.go` suffixes) plus an extension allowlist that includes `.go .ts .tsx .yml .yaml .sh .sql` (`hooks/gitutil.py:524-531`, `:565-575`, `:645-660`). `frontend/e2e/*.spec.ts` is therefore reviewed like any source file. That explains the e2e-only runs.
- **Path filters for the pattern layer:** custom rules in `.claude/security-patterns.{yaml,yml,json}` accept `paths` and `exclude_paths` globs (`hooks/extensibility.py:234-241`; docs page above), but these are additive reminders and "built-in patterns cannot be disabled" individually (`extensibility.py:31-32`).
- **Debounce / throttle:** none for Stop beyond the 3-fire loop cap and 120 s TTL. Commit review has the per-session hourly cap. Four reviews in 7 minutes on the same files (clerk-setup.sh, 2026-10-03T08:05 to 08:24Z) are four separate commit/amend Bash calls, each of which is reviewed.
- **Dedupe by diff hash:** exists but only for the SubagentStop-then-Stop pair (`security_reminder_hook.py:2043-2046`, set at `:2123`); the log shows 103 "diff already reviewed by SubagentStop" skips. The commit-review path dedupes findings by (file, category) and one tool-use id (`:761+`), not by diff hash.
- **On-demand-only mode:** not a built-in mode. Approximation: `ENABLE_CODE_SECURITY_REVIEW=0` (all LLM reviews off) keeps only pattern reminders, and you run `/security-review` by hand.
- **Org rules:** `.claude/claude-security-guidance.md` is appended to the Stop review prompt (README:69-90); the README says the agentic commit reviewer does not read it (README:77), but `review_api.py:173` appends `extensibility.guidance_block()` to the agentic prompt. Source and README disagree here; the source is newer.

### 1c. Docs-versus-source disagreements found

- Docs say default model "Claude Opus 4.7" (`.../security-guidance` "Usage cost"); source and README say "newest Opus", pinned fallback `claude-opus-5-5` (`llm.py:138`). Prefer source.
- Docs say commit/push reviews are "capped at 20 per rolling hour"; source keys the cap per `(session_id, "CommitReview")` (`security_reminder_hook.py:344-352`), so it is per session, not global.
- Docs say Python 3.7+, README says 3.8+ (not relevant to the decision).

## 2. Alternatives

### 2a. On-demand `/security-review`

Official docs: "Analyze the changes on your current branch for security vulnerabilities. Reviews the diff between your branch and origin's default branch ... Needs an `origin` remote" (`https://code.claude.com/docs/en/commands`, command table). The action repo README says it can be customised by copying `security-review.md` into `.claude/commands/` (`anthropics/claude-code-security-review` README:143-148). Cost: one review per invocation, entirely manual. Misses anything you forget to run it on.

### 2b. PR-time CI: `anthropics/claude-code-security-review`

Source read: `action.yml` and `README.md` at commit `0c6a49f1fa56a1d472575da86a94dbc1edb78eda` (2026-02-11, repo not archived).

- **Inputs** (`action.yml:5-49`): `claude-api-key` (required), `comment-pr` (default true), `upload-results` (true), `exclude-directories` (comma-separated directories), `claudecode-timeout` (20 min), `claude-model`, `run-every-commit` (false), `false-positive-filtering-instructions`, `custom-security-scan-instructions`.
- **Triggers:** pull requests only; on other events the step exits with "ClaudeCode only runs on pull requests, skipping" (`action.yml:201-204`).
- **Cost control:** by default it reviews each PR once, via a cache marker `.claudecode-marker` keyed on repository and PR number (`action.yml:76-84`, `:97-113`); `run-every-commit: true` re-reviews every push, and the input description warns of more false positives (`action.yml:36-39`). Path scoping is done by the workflow's `paths:` filter (GitHub feature) and `exclude-directories` (directories only, `github_action_audit.py:138-155`).
- **Model:** `action.yml` leaves `claude-model` empty; the Python default is `claude-opus-4-1-20250805` (`claudecode/constants.py:8`), which agrees with README:57. An empty input falls through `CLAUDE_MODEL` to that default.
- **Permissions:** README quick start uses `pull-requests: write` and `contents: read` (README:21-23); the action authenticates to GitHub with `github.token` (`action.yml:180-181`).
- **Secret needed:** `claude-api-key`, "needs to be enabled for both the Claude API and Claude Code usage" (README:53). The repo has no `ANTHROPIC_API_KEY` secret today (`grep -rn ANTHROPIC .github` is empty), and API billing is separate from a Claude subscription (billing statement is unverified).
- **Hardening caveat:** README:45 says "This action is not hardened against prompt injection attacks and should only be used to review trusted PRs." A solo repo with fork PRs skipped is acceptable; a public repo with external contributors would need the "require approval" setting.
- **Pinning:** the repo publishes no tags or releases (`gh api .../tags` and `.../releases` are empty), so the only immutable ref is a commit SHA. README quick start uses `@main`.

Existing repo CI conventions (`.github/workflows/*.yml`): third-party and GitHub actions pinned to major tags (`actions/checkout@v7`, `golangci/golangci-lint-action@v9`, `neondatabase/create-branch-action@v5`); `persist-credentials: false` on every checkout; top-level `permissions: contents: read` (or `{}` with per-job grants in `preview-environments.yml:30`); `paths:` filters on triggers; `concurrency` used in `preview-environments.yml:24`; local composite actions under `.github/actions/`. Note the plugin's own finding on 2026-09-21 flagged replacing SHA pins with `@v9` (transcript `31cd43e9-...jsonl`), so a SHA pin for the new third-party action matches the stricter side of that decision.

### 2c. Project-level override that disables only the plugin's Stop hook

Not possible. Plainly:

- Hooks merge, they do not override. Docs ("Hook locations"): "Hook entries merge across settings levels rather than replacing each other" and "All matching hooks run in parallel"; "A plugin's or skill's copy of the same handler stays separate" (`https://code.claude.com/docs/en/hooks`). Redefining a `Stop` hook in `.claude/settings.json` adds a hook, it does not replace the plugin's.
- "There is no way to disable an individual hook while keeping it in the configuration" (same page, "Disable or remove hooks").
- `disableAllHooks: true` disables every hook (also a custom status line and `@` file suggestion, `https://code.claude.com/docs/en/settings-reference#disableallhooks`). In this repo that would also kill the Serena hooks and `.claude/hooks/block-commit-on-main.sh` in `.claude/settings.json`, so it is not acceptable.
- `enabledPlugins: {"security-guidance@claude-plugins-official": false}` turns the whole plugin off per scope (`.../settings-reference#enabledplugins`), losing the pattern reminders and commit review too. `defaultEnabled` and plugin `settings` (only `agent`, `subagentStatusLine`) cannot carry per-hook flags (`https://code.claude.com/docs/en/plugins-reference`, `settings` field).
- What does work is the plugin's own env switch. Docs for the plugin list it: "`ENABLE_STOP_REVIEW=0` Disable the end-of-turn diff review" (`https://code.claude.com/docs/en/security-guidance`, "Disable or uninstall"). Settings `env` is documented as "Set environment variables for every session and its subprocesses" (`https://code.claude.com/docs/en/settings-reference#env`), and the hooks page says a hook process "inherits the parent environment" (hooks page, Command hooks). Together these imply the Stop hook sees the variable; I did not run the restart test (see open items).
- Caveat from the settings page: most `env` values in the shared project file apply only after the folder is trusted (`https://code.claude.com/docs/en/settings`, "The key waits for trust").
- The hook script still spawns on every Stop and the baseline is still captured on each prompt; with the switch set it exits before any model call (`security_reminder_hook.py:1981-1984`). Cost: a few hundred milliseconds of git work, no tokens.

## 3. Recommendation for this repo

### 3a. The change (`.claude/settings.json`, shared file)

Add a top-level key; leave the existing `hooks` and `enabledPlugins` untouched:

```json
"env": {
  "ENABLE_STOP_REVIEW": "0"
}
```

Verify after restarting Claude Code: run a turn that edits a file, then `grep "ENABLE_STOP_REVIEW=0" ~/.claude/security/log.txt` should show `Stop hook: ENABLE_STOP_REVIEW=0` (logged at `security_reminder_hook.py:1982`). Optional cost knobs for the commit review, not needed to start: `SG_AGENTIC_MODEL` for a cheaper model, `MAX_COMMIT_REVIEWS_PER_HOUR` for a lower cap.

### 3b. PR-time backstop (optional, `.github/workflows/security-review.yml`)

Validated with `yamllint -c .yamllint.yaml` (one real finding: the SHA-pinned `uses:` line is 93 columns, which cannot be wrapped, so an inline `# yamllint disable-line rule:line-length` is used and `.yamllint.yaml` is left unchanged):

```yaml
name: Security review

# PR-time backstop for the paths where a miss is costly. Runs only when a PR
# touches one of them; the Claude Code security-guidance plugin's per-commit
# review stays on locally. Needs the ANTHROPIC_API_KEY repository secret.

on:
  pull_request:
    branches: [main]
    paths:
      - ".github/workflows/**"
      - ".github/actions/**"
      - "scripts/**"
      - "backend/internal/adapters/http/auth/**"
      - "backend/internal/core/config/**"

concurrency:
  group: security-review-${{ github.event.pull_request.number }}
  cancel-in-progress: true

permissions:
  contents: read
  pull-requests: write

jobs:
  review:
    name: Security review
    runs-on: ubuntu-latest
    # Fork PRs get no repository secrets, so skip them instead of failing.
    if: github.event.pull_request.head.repo.full_name == github.repository
    steps:
      - name: Checkout code
        uses: actions/checkout@v7
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          fetch-depth: 2
          persist-credentials: false

      # The action publishes no tags or releases, so pin the commit.
      # 0c6a49f is main as of 2026-02-11.
      - name: Claude Code security review
        # yamllint disable-line rule:line-length
        uses: anthropics/claude-code-security-review@0c6a49f1fa56a1d472575da86a94dbc1edb78eda
        with:
          claude-api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          run-every-commit: true
          claudecode-timeout: "15"
```

Design notes: `run-every-commit: true` is needed because the default first-run-only marker would skip later pushes to the same PR (which is how the second and third workflow fixes would slip through); `concurrency` with `cancel-in-progress` plus the `paths:` filter bound the cost to sensitive-path pushes; checkout settings match the other workflows (`persist-credentials: false`; the action uses `github.token`, so it should not need persisted credentials, which is unverified end to end). `backend/internal/core/config` is the backend config package (`ls backend/internal/core`). The workflow was not run, only linted and parsed.

### 3c. Options against the 3 real findings

Evidence for attribution: findings 1 to 3 are the only commit-review sessions with findings (per the task brief; finding 1 verified in transcript `31cd43e9-...jsonl`, 2026-09-21T02:24:55Z). The Stop review also flagged the finding 1 diff at 2026-09-21 09:25:22 local (`log.txt`, "LLM code review found 1 high/critical", 14 s after the commit review started). After 2026-09-21 the log shows no Stop finding at all.

| Option | Finding 1 (backend-ci.yml SHA-pin to tag) | Finding 2 (preview-environments.yml action injection) | Finding 3 (scripts/clerk-setup.sh secret exposure) | Cost / gaps |
|---|---|---|---|---|
| Current (Stop + commit/push + patterns) | Caught twice | Caught by commit review | Caught by commit review | About 231 Stop calls plus about 63 agentic sessions |
| **Recommended: `ENABLE_STOP_REVIEW=0`** | Caught by commit review | Caught by commit review | Caught by commit review | Drops Stop calls (all clean after 09-21). Uncommitted work is not reviewed until commit. Commits from your own shell or `!` are not reviewed |
| `ENABLE_COMMIT_REVIEW=0`, Stop on | Likely caught (Stop found it) | Likely missed (no Stop finding after 09-21; inference, unverified) | Likely missed (same) | Opposite trade-off; not recommended |
| Plugin LLM reviews off, patterns only | Reminder on edit (workflow reminder fires on any `.github/workflows/*.yml` edit, `patterns.py:30-70`); no detection | Reminder lists the exact injection shapes, no detection | No built-in secret pattern (`patterns.py` has no secret rule) | Zero model cost; a reminder is not a finding |
| `/security-review` only | Only if you remember to run it | Same | Same | One review per run; needs `origin` |
| CI action only (workflow above, `run-every-commit: true`) | Caught at PR time | Caught at PR time | Caught at PR time (`scripts/**` in `paths:`) | API-key billing, PR-time latency, not hardened against prompt injection, nothing before the push |
| Recommended plus CI backstop | Caught | Caught | Caught | Two reviews on the same sensitive diffs; value is for non-Claude commits |

Trade-off summary: the Stop review fires on every turn that touches a file (frontend e2e specs included) and has no path filter, while the commit/push review is already once per commit and is where the log shows all the post-09-21 yield. Disabling Stop trades earlier feedback (before commit) for lower cost; the pattern reminders still give immediate feedback on edits to workflows.

## 4. Unverified / open

- Restart test of `ENABLE_STOP_REVIEW=0` via `.claude/settings.json` `env` was not run: docs imply it (settings `env` plus hook environment inheritance) but no primary source states that hook subprocesses receive settings `env` values. Check `~/.claude/security/log.txt` for the `ENABLE_STOP_REVIEW=0` line.
- Whether findings 2 and 3 were delivered by the commit review and not also by Stop: transcripts show agentic sessions at those times; the log has no Stop finding after 2026-09-21. Finding 1 was reported by both. Timezone of `log.txt` is WIB (+0700) per `date`; matching to UTC transcripts assumes that.
- Whether the "63 sessions" include push-sweep sessions as well as commit sessions (both use the same prompt, `security_reminder_hook.py:1796`); not split out.
- Whether subagent (worktree) Bash commits trigger the plugin's commit review: hooks.json registers PostToolUse for Bash with no subagent restriction, but I did not confirm hook firing for subagent tool calls.
- API billing separation between an Anthropic API key and a Claude subscription, and that fork PRs receive no secrets (standard GitHub behaviour, not fetched in this session).
- The action's `persist-credentials: false` compatibility and the actual per-run cost of the Opus default model were not tested.
- The two Stop findings on 2026-09-18 (22:29 and 23:41 local, plugin 2.0.8 in the log) are not attributed to any of the 3 reported findings.
