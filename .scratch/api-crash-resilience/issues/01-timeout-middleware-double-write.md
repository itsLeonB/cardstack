# 01: Timeout middleware must not write a second response

**What to build:** A request that outlives the API's request deadline, or whose client disconnects mid-response, stops being able to kill the API process. The timeout wrapper applies the deadline to the request context and never writes a response of its own, so exactly one goroutine owns the HTTP response: the handler, whose error path already maps failures through the single Huma error seam.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

## Why

`sentinel-go@v0.3.4` (the newest published version) runs the handler in a goroutine behind a `timeoutWriter` whose `timedOut` and `wrote` fields are plain, unsynchronized bools. When the request context is done it sets that flag and calls `WriteError(w, 503, …)` on the **raw** `http.ResponseWriter` rather than on the wrapper, while the handler goroutine may still be writing the same response. Two goroutines then mutate the response header map, and the Go runtime kills the process with `fatal error: concurrent map iteration and map write` (stack head: `net/http.Header.Clone` inside `http.Response.WriteHeader`).

The request context is done on a client disconnect as well as at the deadline, so an abandoned browser navigation is enough. It is a runtime fatal, not a panic, so the recovery middleware cannot intercept it.

Evidence, reproduction rate and the 5-second repro command: `.scratch/test-audit/report.md` section 6b.

Two fatal signatures come from the same call: `fatal error: concurrent map iteration and map write` when the second writer is inside `http.Response.WriteHeader` (`Header.Clone`), and `fatal error: concurrent map writes` when it is inside `sentinel-go`'s `WriteJSON` (`Header.Set`). Both were observed during ordinary e2e runs, four crashes in one working session with roughly twenty minutes between them, so the fix has to cover the double-write path rather than one stack shape.

## Acceptance criteria

- [ ] A burst of 1000 requests that abort after 2 ms leaves the API process alive, repeated until the loop has run at least five times without a crash.
- [ ] Neither fatal signature reappears when the e2e suite runs repeatedly against the stack: `concurrent map iteration and map write` and `concurrent map writes`.
- [ ] A request that exceeds the deadline gets exactly one response, carrying the documented timeout failure rather than an unclassified 500; the status and any stable code are declared where the API documents its error codes, and `openapi.json` reflects them.
- [ ] A client that disconnects mid-response produces a logged, classified failure and no second write.
- [ ] No dependency upgrade is required, and the middleware chain keeps its current order and its security headers.
- [ ] `scripts/verification/backend.sh` passes, including the HTTP route suites.
