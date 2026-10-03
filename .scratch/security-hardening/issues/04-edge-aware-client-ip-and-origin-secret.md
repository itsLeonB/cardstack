# 04: Edge-aware client IP and origin secret

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** When the API runs behind Cloudflare with an edge secret configured, it takes the client address from Cloudflare's header, ignores client-supplied address headers, and rejects any request that did not come through the edge (the health check excepted). When the secret is unset, behavior is unchanged, so local development and preview environments keep working. The existing loose per-IP limit stays as a backstop and now keys on the trustworthy address.

**Blocked by:** None (can start immediately). Real-world verification needs ticket 02.

**Status:** ready-for-agent

- [ ] With the edge secret configured, a request without the correct secret header is rejected (401 or 403, a fixed generic body), except the health check, which stays reachable.
- [ ] With the secret configured, the client address comes from Cloudflare's connecting-IP header; `X-Real-IP` and `X-Forwarded-For` from the client are ignored, so a spoofed header cannot change which bucket a request counts against.
- [ ] With the secret unset, no edge check runs and the client address logic is unchanged from today.
- [ ] The secret comparison is constant-time.
- [ ] The existing per-IP limiter still works and keys on the address chosen above.
- [ ] Route tests cover: secret set and correct, secret set and missing, secret unset, health exempt, spoofed address headers ignored.
- [ ] The deployment doc lists the new setting and says it must be left unset in preview environments.
- [ ] Backend build, vet and tests pass.

## Comments
