# Git conventions

## Commit naming

`<type>(<component>): <message>`

Example: `feat(backend): add users api`

## Branch naming

`<type>/<name>`

Example: `feat/implement-users-management`

The convention applies only where the caller can choose the name: the feature branch, and a worker's branch when the harness lets the orchestrator name it. A harness that names the worker's branch itself is fine, because the worktree path is the contract (see [Record the worktree path](agents/workflows/implement.md#delegating)), not the name. A fix-up worker has no branch naming requirement.
