# 10: Guest lock API

**Parent:** `.scratch/security-hardening/spec.md` (absorbs the old ticket 24, `.scratch/cardstack-mvp/issues/24-guest-catalog-preview.md`)

**What to build:** The public catalog endpoints give a Guest a deliberate preview and answer everything beyond it with a stable machine-readable signal. A Guest can browse Series and Expansion Sets, search by name and see one page of 24; anything more returns 401 `login_required`. A signed-in caller gets today's full behavior.

**Blocked by:** 07.

**Status:** done

- [x] For a Guest, the list and name-search endpoints return at most one page; a requested page size above 24 is reduced to 24; any page after the first returns 401 `login_required`.
- [x] For a Guest, a request with a rarity, category or tag filter returns 401 `login_required`; filtering by Expansion Set stays open.
- [x] For a Guest, the facets endpoint returns 401 `login_required`.
- [x] A Guest's result metadata still reports the true total.
- [x] The series, rarities, categories and tags lists stay open to Guests.
- [x] `login_required` is a classified error in the existing error taxonomy of ADR-0013 with a stable code in the response body, documented in the OpenAPI file.
- [x] A signed-in caller's behavior is unchanged, including the page size ceiling of 100.
- [x] A present-but-invalid token is still 401 by ticket 07's rule, not treated as a Guest.
- [x] Route tests cover every rule above for Guest, signed-in and invalid-token callers; the OpenAPI file and generated frontend client are regenerated; backend build, vet and tests pass.

## Comments

### Implementation notes

- The lock lives in `CatalogService` (`SearchCards`, `ListFacets`), driven by a new `Guest` field on `dto.CardFilter` that the catalog handler sets from `authpkg.CallerFrom(ctx).IsGuest()`. The zero value is a Guest, so a caller that forgets to set it gets the locked preview, never the full catalog. Inventory has its own service methods and ignores the field.
- A Guest's page size is `min(requested, 24)`; the metadata still reports the true total because the repository counts before paginating. `localId`, name and Expansion Set filters stay open.
- `login_required` is `apperr.WithCode(ungerr.UnauthorizedError(...), apperr.CodeLoginRequired)`. The Huma error seam (`httpapi.ErrorModel`) writes it as `code` in the body, and the field is in the OpenAPI error schema and the regenerated frontend `ErrorModel`. See ADR-0013.
- Route tests are in `routes/guest_lock_routes_test.go`; branch-level rules are unit-tested in `catalog_service_test.go`, the seam in `errors_test.go`.
