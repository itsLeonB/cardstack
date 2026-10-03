# 10: Guest lock API

**Parent:** `.scratch/security-hardening/spec.md` (absorbs the old ticket 24, `.scratch/cardstack-mvp/issues/24-guest-catalog-preview.md`)

**What to build:** The public catalog endpoints give a Guest a deliberate preview and answer everything beyond it with a stable machine-readable signal. A Guest can browse Series and Expansion Sets, search by name and see one page of 24; anything more returns 401 `login_required`. A signed-in caller gets today's full behavior.

**Blocked by:** 07.

**Status:** ready-for-agent

- [ ] For a Guest, the list and name-search endpoints return at most one page; a requested page size above 24 is reduced to 24; any page after the first returns 401 `login_required`.
- [ ] For a Guest, a request with a rarity, category or tag filter returns 401 `login_required`; filtering by Expansion Set stays open.
- [ ] For a Guest, the facets endpoint returns 401 `login_required`.
- [ ] A Guest's result metadata still reports the true total.
- [ ] The series, rarities, categories and tags lists stay open to Guests.
- [ ] `login_required` is a classified error in the existing error taxonomy of ADR-0013 with a stable code in the response body, documented in the OpenAPI file.
- [ ] A signed-in caller's behavior is unchanged, including the page size ceiling of 100.
- [ ] A present-but-invalid token is still 401 by ticket 07's rule, not treated as a Guest.
- [ ] Route tests cover every rule above for Guest, signed-in and invalid-token callers; the OpenAPI file and generated frontend client are regenerated; backend build, vet and tests pass.

## Comments
