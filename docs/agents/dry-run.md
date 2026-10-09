# Dry run (temporary)

> **Temporary file.** It exists only to prove the setup works. Once every check below passes and you are happy with the setup, delete it with `git rm docs/agents/dry-run.md`. Nothing links to it, so deleting it breaks nothing.

A manual check that the orchestration workflow works in a live T3 Code Nightly session. The static checks cannot prove these: each depends on T3 or Serena behavior at runtime. Do them once after setup.

Use a throwaway task: add a comment to one file in one component.

1. **Hook on a protected branch.** In the orchestrator thread on a protected branch, ask for a commit. The `block-commit-on-main` hook must refuse it with the "create a feature branch" message. Then switch to a feature branch.
2. **Launch.** Ask the orchestrator for the throwaway task as a big task. It must call `orchestrator_capabilities`, then `t3_thread_launch` with a worktree on a child branch, the component model, `full-access`, and the launch message.
3. **Serena in the worktree.** In the child thread, check its first code read goes through `mcp__serena__*` and that its Serena is rooted at the worktree (`git rev-parse --show-toplevel` matches). A built-in code read may draw one `PreToolUse:Read hook error` from `serena-hooks remind`; the retry goes through.
4. **Bootstrap and verification.** The child runs `scripts/bootstrap-worktree/<component>.sh` then `scripts/verification/<component>.sh`, and both exit 0.
5. **Report.** The child commits, never pushes, and reports with `t3_thread_send`. The orchestrator wakes on the report.
6. **Scope check and review.** The orchestrator's scope check passes on a clean worktree, and a deliberate stray edit outside the component directory makes it fail.
7. **Advisor.** Ask the orchestrator to consult the advisor. It launches a plan-mode thread on the advisor model, and the reply opens with `APPROVE`, `CHANGES` or `BLOCK`.
8. **Cleanup.** After the merge the orchestrator archives the child thread, removes the worktree and deletes the branch with `git branch -d`.

Record anything that differs from this list as a fix to `docs/agents/orchestration.md`, in the same change.
