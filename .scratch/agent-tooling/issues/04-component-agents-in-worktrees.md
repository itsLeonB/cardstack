# 04: Component agents launched into worktrees

**What to build:** On a big task the orchestrator launches a backend thread and a frontend thread, each in its own worktree with its own Serena rooted there, and each reports back in a fixed shape. The orchestrator checks scope, merges each branch into the feature branch and cleans up. The instructions describe exactly this and nothing that contradicts it. See `.scratch/agent-tooling/spec.md`, Implementation Decisions "Launch", "Roles", "Reporting, questions and failure", "Review, merge and cleanup" and "Serena policy". Failure recovery and the review-fix loop belong to ticket 06.

**Blocked by:** 02 (both tickets edit the workflow doc).

**Status:** ready-for-agent

Work on the same branch as ticket 02 or a branch from it.

- [ ] The workflow doc's big-task section is rewritten around `t3_thread_launch`: a new worktree based on the feature branch, an explicit branch named `<type>/<feature>-<component>` with the nested-ref ban, no fetch from origin, the derivation of the worktree path from the branch name, Sonnet at medium effort with the model identifier looked up at launch, full-access runtime mode, and the rule to commit everything on the feature branch before launching.
- [ ] The workflow doc has the launch-message template: role pointer (role file, general conventions, component conventions), task or ticket, the orchestrator's thread ID, branch and worktree path, scope, verification commands, commit format, no push, no self-archive, and the report instruction.
- [ ] The report protocol is documented: a queued message to the orchestrator in four parts (status, commit SHAs, verification results, deviations or questions); a blocking question goes the same way and the child ends its turn; the orchestrator answers from the ticket and ADRs first and asks the maintainer only if they don't cover it.
- [ ] The orchestrator waits on each child with the wait tool (30-minute timeout) as well as for its report, and checks scope by diffing changed file names against the component directory. It never edits component code itself during a big task.
- [ ] The ordering rule for API-contract changes, commit naming, branch naming, the architecture review, push confirmation and the orchestrator's ownership of non-component edits are kept.
- [ ] After merging a child branch the documented cleanup order is: archive the thread, remove the worktree, delete the branch with a safe delete.
- [ ] Both component agent files lose the Serena restriction and the built-in-tools instruction, point at the Serena policy doc, and gain the report format, the "ask the orchestrator, never the advisor" rule and the no-self-archive rule. Their frontmatter model stays as the documented default.
- [ ] The Serena policy doc's "who uses Serena" section says everyone in a T3 thread uses Serena, rooted at its worktree for children; the research-pending paragraph is replaced by a pointer to the research notes. The Serena setup notes stay accurate for pi. The project instruction file's Serena-exception line is updated to match.
- [ ] Pre-check: a grep for `root agent only`, `disallowedTools`, `isolation: "worktree"`, `Agent tool`, `Agent-tool` and `plain Claude Code` across the project instruction file, the agent docs folder and the agent files finds no wording that contradicts the spec (a mention in a "not supported" sentence is fine).
- [ ] Docs state rules, not upstream issue numbers. Markdown is not hand-wrapped.
