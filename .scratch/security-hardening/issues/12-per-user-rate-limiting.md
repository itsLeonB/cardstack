# 12: Per-user rate limiting

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** Signed-in callers are rate limited per user inside the API, with tighter limits on the most expensive routes. A caller who exceeds a limit gets a clear 429 with a retry hint, and one account cannot overload the API even from many addresses. Per-IP limiting at the edge and the existing in-API per-IP backstop are untouched.

**Blocked by:** 07.

**Status:** ready-for-agent

- [ ] A limiter keyed by the verified user identity applies to all private routes; name search and facets have their own tighter buckets.
- [ ] Exceeding a limit returns 429 with a Retry-After hint and a fixed generic body, classified through the existing error taxonomy.
- [ ] Limits are configuration with documented defaults (chosen and recorded in this ticket), and the in-memory limiter's ceiling (resets on deploy, single replica) is noted in the code and the deployment doc.
- [ ] One user's traffic does not consume another user's allowance.
- [ ] Guest traffic is not limited by this ticket (the edge and the existing per-IP backstop cover it).
- [ ] Route tests cover: under the limit, over the limit, tighter buckets on search and facets, two users isolated, limit recovery after the window; the arithmetic of the limiter gets unit tests beside it. Backend build, vet and tests pass.

## Comments
