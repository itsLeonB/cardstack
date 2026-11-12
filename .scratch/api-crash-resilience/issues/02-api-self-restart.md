# 02: The API restarts itself after a crash

**What to build:** When the API process dies, the compose stack returns to a serving API without a human running `docker compose up`. After a crash the container comes back on its own, and the stack's readiness signal tells the truth again instead of reporting a permanently unhealthy sidecar next to a dead service.

**Blocked by:** None (can start immediately).

**Status:** done

## Why

The stack runs the API with `restart: "no"`, so a fatal error leaves the container exited. `api-ready` then stays unhealthy forever and every later request fails, which is what turned one crash into a whole e2e session of `Failed to fetch` failures. Recovering meant finding the dead container and starting it by hand.

Ticket 01 removes the crash found in this investigation. This ticket bounds the blast radius of the next one: any fatal in the API, from any cause, must not leave the stack dead.

Production differs: the API runs on Railway, which restarts a crashed container. Compose is where the gap is, and it is also the environment agents and CI use.

Evidence and the reproduction command: `.scratch/test-audit/report.md` section 6b.

## Acceptance criteria

- [x] Killing the API container out from under a running stack brings it back to serving without any manual command, and the stack's readiness reports the API as healthy again once it serves.
- [x] A crash loop is visible rather than silent: repeated restarts are bounded, and the failure is reported where a human or agent will look, not only in container logs.
- [x] The policy covers the API and states what the other services keep, so a deliberate one-shot service is not turned into a restart loop.
- [x] `scripts/verification/frontend.sh e2e` passes against the stack after a forced crash of the API.
- [x] Deployment documentation states the production equivalent, so nobody assumes the compose policy protects production.

## Comments

Implemented on `fix/api-self-restart`: `restart: "on-failure:5"` on the `api` service in `compose.yaml`, with comments stating that every other service keeps the default (migrate stays one-shot), and the production equivalent (Railway restarts the container; the compose policy is dev/CI only) in `docs/agents/deployment.md`. `docs/agents/testing.md`'s `api` row now tells the reader where a crash loop shows: `api-ready` unhealthy, `docker compose ps` showing the exited container, `docker compose up --wait` failing loudly.

Verified live against the stack:

- Ticket 01's fix removed the sentinel-go crash, so the concurrent-curl reproduction no longer kills the API; a forced in-container crash (memory-limited container driven past its limit) restarted the API automatically with no manual command, and `api-ready` reported healthy again once it served.
- A same-image test container failing from inside restarted exactly 5 times and then stayed exited: the loop is bounded.
- `scripts/verification/frontend.sh e2e` against the stack after that forced crash: 89 passed (1.5m).
- A manual `docker compose kill api` does not restart the container: Docker ignores restart policies after operator stop/kill, by design. The compose comment records the distinction (a graceful stop stays stopped).
