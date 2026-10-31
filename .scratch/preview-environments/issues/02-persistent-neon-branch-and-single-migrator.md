# 02: Persistent Neon branch, single migrator, queueing concurrency

**What to build:** A preview's data survives pushes. The PR's Neon branch is created from production only when it does not exist, reused by every later push and by a reopened PR while it still exists, and deleted when the PR closes. The workflow stops running migrations itself, so Railway's pre-deploy migration is the only migrator and runs against the already-configured preview database. Provisioning runs queue per PR instead of cancelling each other, while closing a PR still cleans up. Part of the first pull request.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Opening a PR creates `preview/pr-<number>` from production when it does not exist; a later push reuses it unchanged and never deletes it.
- [ ] A reopened PR reuses its branch if it still exists and recreates it only when it is missing.
- [ ] Closing or merging a PR deletes the branch, as today.
- [ ] The workflow's database migration step, and the Go setup and checkout that existed only for it, are gone. The Neon password is no longer exposed to any step that runs repository code.
- [ ] Concurrency no longer cancels an in-flight provisioning run for a newer push; runs for one PR queue. A closing event is not blocked behind a stale provisioning run.
- [ ] The workflow header comment matches the new behaviour.
- [ ] The deployment guide's preview section describes the persistent branch, how to refresh a preview (delete the `preview/pr-N` branch in the Neon console and push), and the fix for a failed preview migration after a rebase or an edited migration (delete the branch and push).
- [ ] The workflow passes `yamllint`.
