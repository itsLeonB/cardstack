# 04: The stack boots from production images

**What to build:** one command brings up a working database and API for the current worktree, from the unchanged production backend image, with no host Postgres and no ports published. This ticket sets the conventions every later compose ticket follows.

**Blocked by:** None (can start immediately).

**Delivery:** PR 2, branch `chore/compose-stack`, orchestrator-owned (the compose file sits at the repository root, outside the component directories).

**Status:** done

- [x] A compose file at the repository root defines a database on the pgvector image the tests already require, a one-shot migration that runs the production image's migrate binary, and the API running the unchanged production image. The production backend image is not edited.
- [x] A health check gates start-up, so that waiting for the stack returns only when the database is ready, the migration finished and the API answers.
- [x] The compose project name derives from the worktree directory, so two worktrees can run their stacks at once. No host ports are published by default. A gitignored override file can publish them.
- [x] A seed profile loads the end-to-end fixture into the database.
- [x] Environment values come from the existing env files when present, optional when absent, and the API's allowed client addresses are set for the in-network origins that later tickets add.
- [x] Demo: bringing the stack up in a fresh worktree makes the catalog series endpoint answer from inside the compose network. Tearing it down leaves nothing behind but named caches.
- [x] The compose file passes the repo's YAML linter.
