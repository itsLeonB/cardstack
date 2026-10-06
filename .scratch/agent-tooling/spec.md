# Spec: T3 Code orchestration workflow

**Status:** ready-for-agent

Sources: the grilling session of 2026-10-06 and the research in `.scratch/agent-tooling/research/` (`serena-per-worktree-subagents.md`, `t3-code-worktree-orchestration-sources.md`).

## Problem Statement

The maintainer wants one orchestrator that delegates to a backend agent and a frontend agent, each working in its own git worktree with its own Serena instance rooted at that worktree, plus an independent advisor on a stronger model. Today's instructions describe a plain Claude Code workflow built on the Agent tool, and it cannot deliver this. Inline MCP servers in agent frontmatter never start, so Agent-tool subagents see a Serena rooted at the main checkout. They are told to use the built-in file tools instead, and a read in a worktree returns nothing. The advisor is a native Claude Code tool available to every process, so it cannot be restricted to the orchestrator, given a verdict format, or pinned to an effort level.

The research shows that T3 Code Nightly can deliver the workflow. A thread started with `t3_thread_launch` into a worktree gets its own Serena rooted at that worktree, and it can message the orchestrator in both directions. The instructions have not caught up, and the T3 path has never been exercised end to end, so the maintainer cannot trust it for a real feature.

## Solution

From the maintainer's point of view: every piece of work starts in a T3 thread, which acts as the orchestrator. A small task is done by that one thread directly. A larger task is split across a backend agent and a frontend agent, each launched by the orchestrator as its own T3 thread in its own worktree, each using Serena rooted at its worktree. The orchestrator talks to the maintainer; component agents talk only to the orchestrator. When the orchestrator is unsure it launches an advisor role on Opus, which reads the repo and replies with a verdict. The workflow docs, the agent definitions, the Serena policy and the advisor instructions are rewritten to describe exactly this, and a documented dry run in T3 Nightly proves each part before a real feature depends on it.

## User Stories

1. As the maintainer, I want every task to start in a T3 thread, so that there is one supported entry point and no fallback path to keep correct.
2. As the maintainer, I want a small single-component task done by the one T3 thread directly, so that I don't pay for child threads when the change doesn't need them.
3. As the maintainer, I want the orchestrator to decide big versus small using the existing rule (both components, or one component with a large change), so that routing stays predictable.
4. As the orchestrator, I want to stay on the main checkout on the feature branch, so that Serena is rooted where I work and the existing "never commit on main" hook keeps protecting me.
5. As the orchestrator, I want to switch to a feature branch before anything else, so that worktrees have a base to branch from.
6. As the orchestrator, I want to commit my tickets and doc edits on the feature branch before launching a child, so that the child's worktree contains them, because uncommitted changes are not copied into a new worktree.
7. As the orchestrator, I want one launch recipe using `t3_thread_launch` with a new-worktree strategy based on the feature branch, so that I never use `delegate_task` for component work.
8. As the orchestrator, I want the child branch named `<type>/<feature>-<component>`, so that the existing branch convention holds and no ref clashes with the feature branch.
9. As the orchestrator, I want the recipe to forbid a nested ref such as `<feature branch>/<component>`, so that git never has to hold a branch and a directory of the same name.
10. As the orchestrator, I want the worktree path to be predictable from the branch name, so that I can run git commands against a child's worktree without asking T3.
11. As the orchestrator, I want to launch children with a Sonnet model at medium effort chosen through the launch parameters, so that cost is the same as the old agent definitions and does not inherit the maintainer's high effort setting.
12. As the orchestrator, I want to look up the exact model identifier from the capabilities tool at launch time, so that the recipe never hard-codes an identifier that goes stale.
13. As the orchestrator, I want children launched with full access, so that they can run builds, tests and commits unattended.
14. As the orchestrator, I want one launch-message template, so that every child is told the same things in the same order.
15. As a component agent, I want the launch message to point me at my role file and the code conventions, so that I get my instructions even though T3 does not read agent files.
16. As a component agent, I want the launch message to carry the ticket path, my branch, my worktree path, the orchestrator's thread ID and my verification commands, so that I can work and report without guessing.
17. As a component agent, I want to be told never to push and never to archive my own thread, so that I don't break the merge flow or kill my own run.
18. As a component agent, I want to use Serena for all code reads and edits, so that I get symbol-level tools rooted at my worktree.
19. As a component agent, I want to verify and commit on my own branch with the commit naming convention, so that the orchestrator can review and merge cleanly.
20. As a component agent, I want to report with a fixed four-part message (status, commit SHAs, verification results, deviations or questions), so that the orchestrator can parse it at a glance.
21. As a component agent, I want to send my report by queued message to the orchestrator's thread, so that a busy orchestrator receives it as a new turn after its current one.
22. As a component agent, I want to send a blocking question to the orchestrator and then end my turn, so that I never guess at a decision that belongs to the maintainer or the ticket.
23. As the orchestrator, I want to answer a child's question from the ticket and ADRs first, so that the maintainer is only interrupted for real decisions.
24. As the orchestrator, I want to ask the maintainer when the ticket and ADRs don't answer a child's question, so that I am the single point of contact.
25. As the maintainer, I want to talk only to the orchestrator, so that I never track several threads.
26. As the orchestrator, I want to poll each child with the wait tool and a 30-minute timeout in addition to waiting for its report, so that a child that dies silently cannot hang the run.
27. As the orchestrator, I want to check a child's scope with a diff of changed file names against its component directory, so that the scope limit is enforced even though tool restrictions no longer apply.
28. As the orchestrator, I want to run the `code-review` skill on a child's worktree diff, so that the review step is unchanged.
29. As the orchestrator, I want to send review findings back to the same child thread, so that the child keeps its context, fixes, re-verifies, commits and reports again.
30. As the orchestrator, I want a failed or timed-out child handled by reading its worktree's git state first, so that committed work is never thrown away.
31. As the orchestrator, I want to relaunch a replacement thread into the same worktree with a "continue from this state" prompt when work is uncommitted or missing, so that a dead child costs one retry, not a restart.
32. As the orchestrator, I want to stop and ask the maintainer after a second failure on the same task, so that I don't loop.
33. As the orchestrator, I want to never edit inside the backend or frontend directories myself during a big task, so that the review stays meaningful.
34. As the orchestrator, I want to keep the existing ordering rule (backend first and merged when the API contract changes, otherwise parallel), so that the frontend regenerates its client from the new contract.
35. As the orchestrator, I want to merge each child's branch into the feature branch after review, so that the feature branch holds the integrated result.
36. As the orchestrator, I want a fixed cleanup order after merge (archive the thread, remove the worktree, delete the branch safely), so that nothing is left behind and an unmerged branch is never deleted.
37. As the orchestrator, I want to run the architecture review on the whole feature branch diff after merging, so that cross-component problems are caught.
38. As the orchestrator, I want to confirm with the maintainer before any push, so that outward-facing actions stay under the maintainer's control.
39. As the orchestrator, I want to own every edit outside the component directories (docs, glossary, ADRs, ticket status), so that the component agents stay scoped.
40. As the orchestrator, I want to launch an advisor role on demand, so that I can get an independent view from a stronger model.
41. As the orchestrator, I want the advisor to be one thread per run and reused through messages, so that it keeps context across consults.
42. As the orchestrator, I want to consult the advisor before choosing between approaches, when stuck, before declaring multi-step work done, and when a child's question is not answered by the ticket or ADRs, so that consults happen at the points where they matter.
43. As the orchestrator, I want the advisor's reply to be `APPROVE`, `CHANGES` with a list, or `BLOCK`, each with reasons, so that I can act on it directly.
44. As the orchestrator, I want to send one reconcile message when I disagree with the advisor, and then ask the maintainer if we still disagree, so that disagreements are resolved by a human rather than by whoever speaks last.
45. As the advisor, I want read access to the main checkout, the tickets and the children's worktrees, so that I can judge real code instead of a description.
46. As the advisor, I want to be unable to edit files, so that an advisor can never change the work it is judging.
47. As the maintainer, I want the advisor on Opus at medium effort set at launch, so that the model and effort are enforced rather than hoped for.
48. As the maintainer, I want the native advisor tool disabled for this project, so that there is exactly one advisor mechanism.
49. As the maintainer, I want component agents to ask the orchestrator instead of calling an advisor, so that advisor cost stays under the orchestrator's control.
50. As the maintainer, I want a model and effort table in the workflow docs (orchestrator Sonnet high, component agents Sonnet medium, advisor Opus medium), so that the cost profile is written down in one place.
51. As the maintainer, I want the Serena policy to say that everyone in a T3 thread uses Serena, so that the old "root agent only" rule and the tool restriction that enforced it are gone.
52. As the maintainer, I want the Agent-tool and `isolation: "worktree"` route removed from the workflow docs, so that no agent follows an instruction that cannot work.
53. As the maintainer, I want the always-on Serena reminder hook to keep working for launched threads, so that component agents get the same nudges the orchestrator does.
54. As the maintainer, I want each doc to state rules rather than upstream issue numbers, so that docs don't go stale when a bug is fixed.
55. As the maintainer, I want the upstream issues that shaped these rules recorded in the research note, so that I can recheck them when T3 updates.
56. As the maintainer, I want the dangling references to ticket comments and tickets that were never saved removed, so that nobody chases a section that doesn't exist.
57. As the maintainer, I want a dry-run checklist covering six cases, so that each part of the workflow is proven before a real feature depends on it.
58. As the maintainer, I want the dry run's results recorded in its ticket, so that a failure leaves a record of what broke.
59. As a future maintainer, I want the unverified assumptions listed in one place, so that I know which behaviors the dry run has to confirm.
60. As the maintainer, I want the small-task path to still run verification, a `code-review` pass and a commit on the feature branch, so that small tasks are as safe as they are today.

## Implementation Decisions

**Entry point and routing**
- The project assumes every task starts in a T3 thread. Plain Claude Code is not a supported entry point, and the docs do not describe a fallback for it.
- The big-versus-small rule is unchanged. A small task is done by the single thread directly on the feature branch, in the main checkout, with Serena, with no children and no worktree. A big task uses the component agents.
- "Feature branch first" and the commit-on-main hook are unchanged. The orchestrator thread runs on the main checkout.

**Launch**
- Component work is launched only with `t3_thread_launch`. `delegate_task` is not used for component work.
- Each child gets a new worktree based on the feature branch, an explicit branch named `<type>/<feature>-<component>`, and no fetch from origin. The worktree path is chosen by T3 and derived from the branch name by replacing slashes with dashes. A nested ref under the feature branch is forbidden.
- The orchestrator commits everything on the feature branch before launching.
- Children run Sonnet at medium effort, with full-access runtime mode. The model identifier is looked up from the capabilities tool at launch, never hard-coded.
- The launch message follows one template: role pointer (the role file, the general conventions and the component conventions), task or ticket, the orchestrator's thread ID, branch and worktree path, scope, verification commands, commit format, no push, no self-archive, and the report instruction.

**Roles**
- T3 does not read agent definition files, so the role files stay as the single source of role text and the launch message tells the child to read them. Their frontmatter becomes documentation only.
- The component agents' Serena restriction and the "built-in tools only" instruction are removed. They use Serena under the same policy as the root agent.
- A new advisor role file defines the advisor's instructions and verdict format.

**Reporting, questions and failure**
- Reports and questions go to the orchestrator by queued message. A report has four parts: status, commit SHAs, verification results, deviations or questions. A child that asks a question ends its turn and waits for the reply.
- The orchestrator waits on each child with the wait tool, 30-minute timeout, in addition to the child's own report. T3 gives no completion signal for launched threads, and a child can end failed without any message.
- On failure or timeout the orchestrator reads the thread and the worktree's git state. It relaunches into the same worktree, using the existing-worktree strategy, with a continue prompt. After a second failure on the same task it stops and asks the maintainer. It never edits component code itself.
- Scope is checked by diffing changed file names against the component directory.

**Review, merge and cleanup**
- The review loop is unchanged except that fixes go to the same child thread by queued message instead of an Agent-tool continuation.
- The ordering rule for API-contract changes is unchanged.
- After merging a child branch the orchestrator archives the thread, removes the worktree and deletes the branch with a safe delete, in that order, because T3 has no tool for worktree or branch removal.
- The architecture review, the push confirmation and the orchestrator's ownership of non-component edits are unchanged.

**Advisor**
- The advisor is a role launched through `t3_thread_launch`, one thread per run, reused through messages and archived at the end, started fresh only if its context goes stale.
- It runs on the main checkout in plan mode, with auto-accept-edits runtime mode so reads and diffs do not stall, on Opus at medium effort. It uses Serena for reads. It reads children's worktrees by absolute path.
- The orchestrator consults it before choosing between approaches, when stuck, before declaring multi-step work done, and when a child's question is not answered by the ticket or ADRs. Doc, ticket, config and one-line edits go ahead without it, as today.
- Verdicts are `APPROVE`, `CHANGES` with a list, or `BLOCK`, each with reasons. The orchestrator treats the reply as advice, sends one reconcile message on disagreement, then asks the maintainer.
- Component agents never call the advisor. They ask the orchestrator.
- The native advisor tool is disabled for the project through the project settings environment block (`CLAUDE_CODE_DISABLE_ADVISOR_TOOL=1`). No advisor model is pinned in project settings. The maintainer's user-level advisor setting is left alone and is ignored here.
- The `AGENTS.md` Advisor section keeps the "call when unsure" rule, says only the orchestrator consults the advisor, and points at the workflow doc for the mechanism.

**Serena policy**
- Everyone in a T3 thread uses Serena. The "who uses Serena" section is rewritten, and the research-pending paragraph is replaced by a pointer to the research notes.
- The always-on reminder hook applies to launched threads, because they are top-level threads rather than subagents. No hook change is planned. The dry run confirms the behavior.
- The setup notes for the maintainer stay accurate for pi. The pi subagent files are symlinks to the same agent files, so the Serena policy change reaches them too. pi orchestration is not designed here.

**Docs to rewrite**
- The orchestration workflow doc (launch recipe, launch template, report format, failure policy, review loop, cleanup, model and effort table, advisor recipe, small-task path).
- Both component agent files, plus the new advisor file.
- The Serena tool policy and the Serena setup notes.
- The `AGENTS.md` lines on development orchestration, the advisor and the Serena exception.
- The project settings file (environment block only).
- The research notes: remove dangling references, record the upstream issues (#15136, #15135, #15173, #13490, #15082 on `pingdotgg/t3code`) and the unverified assumptions.
- Ticket 01: the follow-up line about the Serena policy doc is closed by this work.

## Testing Decisions

- **One seam:** an end-to-end orchestration run in a T3 thread on a throwaway branch. The orchestrator follows only the new docs. A good test judges outcomes from outside: git state, thread messages, the merged result, and the worktrees and branches left behind. It does not inspect how the orchestrator reasoned.
- **Six dry-run cases:**
  1. Parallel happy path: a backend and a frontend marker symbol, each added through Serena in its own worktree, verified, committed, reported, reviewed, merged and cleaned up.
  2. Review-fix loop: the orchestrator sends one finding back and the child fixes it.
  3. Silent child death: a child is told to archive itself. The 30-minute wait, the git-state read and the replacement launch are exercised.
  4. Long wait: confirm a long wait by the orchestrator does not trip the 30-minute idle release.
  5. Advisor role: launch, one consult, the reply format, Opus and the effort level confirmed, an edit attempt refused, and the native advisor tool absent.
  6. Small-task path: a single thread implements a marker change and consults the advisor role once.
- **Pre-check, not a second seam:** after the doc edits a grep finds no leftover Serena-restriction, Agent-tool, worktree-isolation or plain-Claude-Code-fallback wording in the instruction files.
- **Prior art:** the 2026-10-06 probes in ticket 01. The repeatable check there is a worktree thread writing a fresh marker symbol, finding it through its own Serena, and editing it so that the worktree changes and the main checkout does not.
- The maintainer runs the dry run in T3 Nightly and records the results in the dry-run ticket. This spec's author works in plain Claude Code and cannot run it.

## Out of Scope

- pi orchestration and pi-specific changes.
- Using `delegate_task` for component work.
- Filing or commenting on upstream issues. The maintainer may comment on #15136 if wanted.
- Running the API-contract-ordered backend-then-frontend chain in the dry run. It is documented, not exercised.
- Any change to backend or frontend code.
- Changing the maintainer's user-level settings or hooks.
- Enforcing the advisor's effort level beyond what the launch parameters provide.
- A mechanism for component agents to call an advisor.

## Further Notes

- **Unverified assumptions the dry run must confirm:**
  - The project environment block disables the native advisor tool inside a launched thread, which relies on T3 loading project settings (the SDK's default when no setting sources are given, per the research note).
  - A launched Claude thread sees the agent files through Claude Code itself. This is irrelevant to the design, which tells the child to read them explicitly.
  - Auto-accept-edits combined with plan mode lets the advisor run read-only shell commands such as a git diff without stalling.
  - `medium` is accepted as the effort value in the launch's model options.
  - A long wait by the orchestrator does not trigger the idle release described in #15173.
  - The reminder hook fires in launched threads.
- **Known upstream bugs that shape the rules:** #15136 (a child dies after a worktree handoff, which explains the earlier failed probe and the stuck task), #15135 (self-archive fails the run, hence "never archive yourself"), #15173 (idle release while waiting on a delegated child), #13490 and #15082 (no parent notification for follow-up turns or for a delegated task that asks a question, hence launching and polling instead of delegating).
- The source commit for the T3 behavior in the research notes is `0ecb78ed0d351b65ef68262f632359223f9c389b`. The installed Nightly may differ.
- Tickets, in order: 02 small task and advisor role, 03 dry run of 02 (cases 5 and 6), 04 component agents in worktrees (blocked by 02), 05 dry run of 04 (case 1), 06 review loop and failure recovery (blocked by 04), 07 dry run of 06 (cases 2 to 4), 08 research notes cleanup (independent). The dry-run tickets are `ready-for-human`.
