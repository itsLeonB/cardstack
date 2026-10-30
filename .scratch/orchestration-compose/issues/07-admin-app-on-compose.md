# 07: Admin app on compose

**What to build:** the Admin app runs in the stack from its own production-style image served by the same approach, and its checks run through the frontend tooling container like the user app's.

**Blocked by:** 06 (User app image, serve script and frontend verification).

**Delivery:** PR 2, branch `chore/compose-stack`. A frontend worker writes the Admin app image and its tests; the orchestrator writes the compose, script and workflow changes.

**Status:** done

- [x] The Admin app has a production-style image kept with its package, with the build arguments its build requires, and it is served by a serve script that reads the Admin app's own `vercel.json`, which sets the no-index header on every path.
- [x] The Admin app's serve script is covered by a request-in, response-out test against the real `vercel.json`, including the no-index header on every path and the fallback to the shell page. Whether the script is shared with the user app or copied is the worker's call, as long as each app reads its own configuration file.
- [x] The compose file gains the Admin app service. The API's allowed client addresses and the Admin allow-list settings in the stack include the Admin app's in-network origin.
- [x] The frontend verification script runs the Admin app's steps as well as the user app's, with the existing behavior of skipping the package where it is absent.
- [x] The Admin app's CI workflow runs the same script per step with its job names unchanged.
- [x] The frontend worker reviews its diff once before reporting. The YAML passes the repo's YAML linter.
