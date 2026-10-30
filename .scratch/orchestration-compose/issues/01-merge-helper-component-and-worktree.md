# 01: Merge helper takes a component and a worktree, and the branch rule

**What to build:** the orchestrator merges and cleans up a worker's branch by naming the component and the worker's worktree path, so it never copies a branch name by hand and never depends on a name the harness chose. The orchestration doc states the rule: the branch name is a request, the worktree path is the contract.

**Blocked by:** None (can start immediately).

**Delivery:** PR 1, branch `docs/orchestration-review-flow`, orchestrator-owned (tools, docs).

**Status:** done

- [x] The scope flag accepts several values after one flag, and a repeated flag still works. The documented syntax now matches the parser.
- [x] A component flag expands a component name to its directories, with the frontend component covering both frontend packages. The component-to-directory map lives only in the helper.
- [x] A worktree flag takes a worktree path, resolves the live branch from git, and refuses a detached HEAD. The existing branch argument still works. The cleanup subcommand accepts the worktree form too.
- [x] The helper's tests build real git repositories and cover: several scope values, a repeated scope flag, the component expansion, the worktree form resolving the live branch (including after the branch was renamed), a detached HEAD being refused, and cleanup by worktree path.
- [x] The orchestration doc's merge and cleanup steps use the component and worktree forms. It says the orchestrator records the worker's worktree path from the harness, reads the live branch from git, and that workers never rename their branch.
- [x] The branch naming convention applies only where the caller can choose the name. Fix-up workers have no branch naming requirement. The commit naming convention is unchanged. No doc names a specific orchestration product.
