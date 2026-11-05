# 02: The API restarts itself after a crash

**What to build:** When the API process dies, the compose stack returns to a serving API without a human running `docker compose up`. After a crash the container comes back on its own, and the stack's readiness signal tells the truth again instead of reporting a permanently unhealthy sidecar next to a dead service.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

## Why

The stack runs the API with `restart: "no"`, so a fatal error leaves the container exited. `api-ready` then stays unhealthy forever and every later request fails, which is what turned one crash into a whole e2e session of `Failed to fetch` failures. Recovering meant finding the dead container and starting it by hand.

Ticket 01 removes the crash found in this investigation. This ticket bounds the blast radius of the next one: any fatal in the API, from any cause, must not leave the stack dead.

Production differs: the API runs on Railway, which restarts a crashed container. Compose is where the gap is, and it is also the environment agents and CI use.

Evidence and the reproduction command: `.scratch/test-audit/report.md` section 6b.

## Acceptance criteria

- [ ] Killing the API container out from under a running stack brings it back to serving without any manual command, and the stack's readiness reports the API as healthy again once it serves.
- [ ] A crash loop is visible rather than silent: repeated restarts are bounded, and the failure is reported where a human or agent will look, not only in container logs.
- [ ] The policy covers the API and states what the other services keep, so a deliberate one-shot service is not turned into a restart loop.
- [ ] `scripts/verification/frontend.sh e2e` passes against the stack after a forced crash of the API.
- [ ] Deployment documentation states the production equivalent, so nobody assumes the compose policy protects production.
