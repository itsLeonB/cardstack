# 12: Per-user rate limiting

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** Signed-in callers are rate limited per user inside the API, with tighter limits on the most expensive routes. A caller who exceeds a limit gets a clear 429 with a retry hint, and one account cannot overload the API even from many addresses. Per-IP limiting at the edge and the existing in-API per-IP backstop are untouched.

**Blocked by:** 07.

**Status:** done

- [x] A limiter keyed by the verified user identity applies to all private routes; name search and facets have their own tighter buckets.
- [x] Exceeding a limit returns 429 with a Retry-After hint and a fixed generic body, classified through the existing error taxonomy.
- [x] Limits are configuration with documented defaults (chosen and recorded in this ticket), and the in-memory limiter's ceiling (resets on deploy, single replica) is noted in the code and the deployment doc.
- [x] One user's traffic does not consume another user's allowance.
- [x] Guest traffic is not limited by this ticket (the edge and the existing per-IP backstop cover it).
- [x] Route tests cover: under the limit, over the limit, tighter buckets on search and facets, two users isolated, limit recovery after the window; the arithmetic of the limiter gets unit tests beside it. Backend build, vet and tests pass.

## Comments

### Implementation notes

- Defaults (token bucket per user per tier, `RATE_LIMIT_*` settings): general 300 a minute with a burst of 100; name search (`GET /catalog/cards`) 60 a minute, burst 30; facets (`GET /catalog/facets`) 30 a minute, burst 15. The general tier leaves room for fast browsing across Collections and Inventory; search allows a sustained page a second of infinite scroll after a burst; facets reload only when a filter changes. The API refuses to boot with any setting at zero or less.
- Search and facets spend only their own bucket, not the general one, so each tier's number is the whole allowance for those routes.
- The limiter (`backend/internal/adapters/http/ratelimit`) wraps `golang.org/x/time/rate` per user id, refuses without spending a token, and sweeps full buckets once a minute so memory follows active users. It runs as Huma middleware after `auth.Guard` on every guarded route group; Guests pass through. A 429 carries `Retry-After` (whole seconds, rounded up) and the fixed detail `too many requests, retry later`, built as `ungerr.TooManyRequestsError` through the error seam, with no `code`, since clients only need the status.
- The in-memory ceiling (resets on deploy, single replica only) is in the package comment and `docs/agents/deployment.md`.
