# 01: Timeout middleware must not write a second response

**What to build:** Repo-owned request-deadline middleware that runs the handler synchronously and never writes a response of its own. A deadline or client disconnect must not trigger a competing response writer or kill the API process. Keep the handler as the sole response owner and classify returned deadline errors through the existing Huma error path.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

## Why

`sentinel-go@v0.3.4` (the newest published version) runs the handler in a goroutine behind a `timeoutWriter` whose `timedOut` and `wrote` fields are plain, unsynchronized bools. When the request context is done it sets that flag and calls `WriteError(w, 503, …)` on the **raw** `http.ResponseWriter` rather than on the wrapper, while the handler goroutine may still be writing the same response. Two goroutines then mutate the response header map, and the Go runtime kills the process with `fatal error: concurrent map iteration and map write` (stack head: `net/http.Header.Clone` inside `http.Response.WriteHeader`).

The request context is done on a client disconnect as well as at the deadline, so an abandoned browser navigation is enough. It is a runtime fatal, not a panic, so the recovery middleware cannot intercept it.

The Gin adapter also discards the wrapped writer (`sentinel-go@v0.3.4/httpserver/adapters/gin/gin.go`), so synchronizing `timeoutWriter` alone would not protect the handler's writes. Returning from the middleware while its handler goroutine still runs also lets Gin reuse its request context before that work ends.

Evidence, reproduction rate and the 5-second repro command: `.scratch/test-audit/report.md` section 6b.

Two fatal signatures come from the same call: `fatal error: concurrent map iteration and map write` when the second writer is inside `http.Response.WriteHeader` (`Header.Clone`), and `fatal error: concurrent map writes` when it is inside `sentinel-go`'s `WriteJSON` (`Header.Set`). Both were observed during ordinary e2e runs, four crashes in one working session with roughly twenty minutes between them, so the fix has to cover the double-write path rather than one stack shape.

## Agreed design

- Replace only the timeout entry in `backend/internal/adapters/http/setup_sentinel.go` with repo-owned middleware. Keep `APP_TIMEOUT`, the current chain order and the security headers. Keep sentinel-go for the other middleware; no dependency upgrade is needed.
- Mirror sentinel-go's timeout setting and integration point, not its execution design. Apply `context.WithTimeout` to the request, defer cleanup, and call the handler on the current goroutine. Add no timeout writer, handler goroutine, response buffer or separate timeout response.
- At the existing Huma error path in `backend/internal/adapters/http/huma/errors.go`, recognize raw and wrapped `context.DeadlineExceeded` and return a redacted HTTP 503 response. Preserve explicit `AppError` classifications and reuse the existing unavailable error type. Use `errors.Is` for standard wrapping; also inspect `ungerr` wrappers through their supported API because `ungerr@v0.5.0` does not expose the standard `Unwrap()` method. Do not classify an unrelated error as a timeout just because the request context expired. Use 503 to match sentinel-go's actual response, not the 504 claimed by its Gin adapter comment. Document the status and regenerate `openapi.json`. Add no new stable error code: no client branching behavior needs one (ADR-0013).
- Classify request cancellation separately from deadline expiry in server logs. Observe the request context before this middleware calls its own cleanup `cancel`, so normal cleanup does not produce a false failure. Include cancellation during response writing even when Huma does not return the write error to the outer middleware, and avoid duplicate failure logs.
- Keep recovery from attempting an error response after the response has started. Give the server write timeout time beyond the application deadline to send a cooperative timeout response; the current equal 10-second defaults do not leave that time. Cover the resulting timeout settings in the issue's tests.

## Deadline contract and limits

A request deadline asks context-aware work to stop; it does not force the handler to return. If a handler returns a deadline error before the response starts and the connection remains writable, the API sends one classified 503 response through Huma.

If a handler ignores cancellation and later succeeds, preserve its success. Do not report a timeout merely because the clock expired, especially after a mutation succeeded. This change does not promise to stop such work, undo a mutation, or send a response by a fixed deadline.

Once a response has started, neither timeout handling nor recovery may replace it with timeout or panic JSON. A client disconnect produces classified server logs, not a promise that the client receives a response. A connection failure or the server write timeout can prevent delivery even when the handler returns a deadline error.

## Acceptance criteria

### Response ownership and classification

- [ ] The repo-owned middleware runs the handler synchronously with a deadline context and never writes its own response. It retains `APP_TIMEOUT`, chain order and security headers without upgrading dependencies.
- [ ] A handler that returns a raw, standard-wrapped or `ungerr`-wrapped `context.DeadlineExceeded` before its response starts produces exactly one redacted 503 response while the connection remains writable. Existing explicit `AppError` classifications remain intact. The API error documentation describes this response and `openapi.json` explicitly lists 503 for operations governed by the request deadline; no new stable error code is added.
- [ ] A handler that ignores cancellation and later succeeds retains its successful response. A started response is never followed by replacement error JSON, including when recovery handles a later panic.
- [ ] Cancellation during a started response produces classified server logs and no second write. Logs distinguish cancellation from deadline expiry, do not duplicate the same failure, and do not report ordinary middleware cleanup as cancellation.
- [ ] The default server write timeout exceeds the application deadline and leaves time to send the timeout response. The supported timeout settings and their delivery limits are documented and tested.

### Regression proof

- [ ] A Gin/Huma boundary suite covers deadline classification (including standard and `ungerr` wrapping and preserved explicit `AppError` classifications), cancellation during a started response, context-ignoring late success, panic after a started response, and normal completion without a false cancellation log. Assert response ownership, classified logs and preserved headers; run it with the race detector through the shared compose test environment. Exercise the actual middleware chain: the existing route suites use `humatest` without `setupSentinel`.
- [ ] A burst of 1000 requests that abort after 2 ms leaves the API process alive, repeated until the loop has run at least five times without a crash.
- [ ] Neither fatal signature reappears when the e2e suite runs repeatedly against the stack: `concurrent map iteration and map write` and `concurrent map writes`.
- [ ] `scripts/verification/backend.sh` passes, including the HTTP route suites.

## Comments

The design interview approved synchronous, context-only middleware after an independent Pi GPT-6 Astra advisor compared it with a hard response cutoff and a synchronized copy of sentinel-go's design. A hard cutoff would require a different response-ownership contract and isolated handler state; it is outside this issue. This decision changes the issue contract only; implementation and verification remain open.
