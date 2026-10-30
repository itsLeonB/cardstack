# 09: Compose-first docs and cleanup

**What to build:** the testing and environment docs describe one way to get an environment, the compose stack, and the stale instructions it replaces are gone.

**Blocked by:** 05 (Backend verification through compose), 07 (Admin app on compose), 08 (Browser suite against the built stack).

**Delivery:** PR 2, branch `chore/compose-stack`, orchestrator-owned (docs, scripts).

**Status:** claimed

- [ ] The testing doc is compose-first: bring the stack up once, then run every tool with `compose exec` or the verification scripts. The manual Postgres provisioning steps and the uuid compatibility shim are removed, or reduced to one short fallback pointer.
- [ ] The environment doc covers the stack's env files and the gitignored override that publishes ports and rebuilds the app images with localhost addresses for a human's browser. It states that the host keeps its language toolchains for Serena's language servers.
- [ ] The orchestration doc's verification section points at the verification scripts as the single entrypoint, including their step names, and notes that an end-to-end run rebuilds the app images while unit tests, lint and typecheck do not need a rebuild.
- [ ] The worktree bootstrap scripts and every doc that mentions installing dependencies by hand match the new flow. No doc still tells an agent to provision Postgres itself.
- [ ] The integration code review has run once on the merged branch, and its findings are fixed.
- [ ] A cold stack start, a single step run, and a rebuild after a source edit have each been run locally from a clean worktree and recorded in the pull request.
