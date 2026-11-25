# Delegated workers

## Parent: send a complete task

Give each worker its goal, acceptance criteria, spec path, scope, checkout, checks and expected handback. A fresh worker sees its task, not the parent's conversation. Never send "follow the parent prompt" as a substitute for requirements. Include any permission to create children as an exact standalone line: `Child delegation: allowed`. Omit that line for ordinary implementation workers.

The global subagent instructions describe how a parent delegates. They do not require a worker to delegate again.

## Worker: preflight and execute

Pi's project extension recognizes the T3 kickoff prefix `Act as the <role> sub-agent for this task.` in the first user message. It checks referenced Markdown paths, records the starting Git state and lists registered Serena discovery/read tools. This is a preflight, not proof of a working MCP connection. Read the supplied task and existing spec; do not ask for a spec that is already named and present.

Use the discovery, project-path checks and fallback in `conventions/serena.md`. Missing initial tool declarations are not proof that a server is unavailable. If a real tool operation fails, record its name and error, then use the supported fallback. If no fallback can satisfy the task, report the precise blocker instead of returning an unsupported tool-access claim.

The extension blocks worker calls to delegate or create more threads, including calls dispatched through Pi's nested tool execution. A permission line in the original task is the only opt-in; a later worker message cannot grant it. This is a workflow guard for Pi/T3 tasks, not an operating-system sandbox or an enforcement mechanism for other harnesses.

## Worker: hand back evidence

An implementation worker ends its report with two fields:

- `Outcome: implemented` — the requested change exists; name the commit or uncommitted files, check commands/results and remaining limits.
- `Outcome: verified-unchanged` — no change was needed; name the inspected behavior and checks that establish this.
- `Outcome: blocked` — a missing decision, permission or failed operation prevents progress; name it and what would unblock it. A tool-access blocker names the failed call and fallback attempt.
- `Outcome: failed` — the task remains incomplete; name the failure and any partial work.

Each outcome requires an `Evidence:` line with the supporting facts. A checkpoint commit is allowed under `testing.md`; a red merge gate must still be reported as red.

Before an implementation worker settles, the extension checks the report format, whether an implemented claim has a changed Git snapshot, and whether a claimed tool blocker names an observed failed call. It requests one corrective continuation at most. If the report still fails validation, it records a visible failed-contract message instead of looping. These checks do not prove semantic correctness or replace review. Aborting a run does not trigger recovery.

## Parent: inspect before accepting or retrying

`task_status` saying `completed` means a run ended, not that the task passed. Read the outcome, inspect the actual diff or commit, and check the reported commands. Missing evidence is a failed handback, not a reason to tell the user the work succeeded.

After the first failed or unsupported implementation handback, read that worker's activity before starting another worker. Find whether it received the brief, used tools, re-delegated or hit a real error. Repair that cause; do not simply lengthen and resend the prompt. Keep the requested provider/model unless the user approves a change.

During one Pi runtime, the extension tracks implementation delegates and blocks a new delegation after an unaccepted result until the parent reads the failed worker's activity. This retry check is advisory across reloads/resumes; the parent must still follow the procedure. It does not alter T3's run status or approve a task automatically.

## Loading and checks

The extension loads from the trusted project's `.pi/extensions` on the next session or `/reload`. A worker worktree needs a commit containing it; uncommitted parent edits do not enter a new worktree. Existing running workers keep their loaded version.

Run `bash scripts/tests/worker-contract/run.sh` for the extension's event-level checks. Conventions CI runs these checks and the existing project extension checks. The tests cover direct/proxied delegation, original-task permission, preflight paths/tools, bounded recovery, changed/unchanged Git states and parent activity inspection. They use Pi's documented callback contract; a live provider run remains a separate smoke check.
