# 03: Request hardening

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The public API stops accepting unbounded input, stops echoing caller input in error bodies, stops advertising its own docs in production, and sends the missing security headers. A caller sees: a long `name` search or an oversized repeated filter is rejected with an ordinary validation error; malformed requests get generic messages; the docs and OpenAPI endpoints are gone in production; every response carries HSTS and a Referrer-Policy header.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] `name` search longer than 64 characters is rejected with a validation error, on every endpoint that accepts it.
- [x] Each repeated filter (Expansion Set, rarity, category, tag) rejects more than 20 values, on every endpoint that accepts them, including facets.
- [x] Error responses for malformed requests no longer include caller-supplied values: invalid-identifier messages and the detail strings Huma produces for decode errors are replaced by fixed messages. ADR-0013's rule is extended to client errors, and the ADR or its Consequences section says so.
- [x] Every API response carries HSTS and a Referrer-Policy header alongside the existing headers.
- [x] In production mode the interactive docs page and the OpenAPI endpoints return 404; in other modes they are unchanged. The committed OpenAPI file and the frontend code generation still work.
- [x] CORS behavior is unchanged, and no cache is added to facets.
- [x] Route tests cover each criterion at the HTTP level; the backend build, vet and tests pass.

## Comments
