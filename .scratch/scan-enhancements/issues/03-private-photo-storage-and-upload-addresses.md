# 03: Private photo storage and upload addresses

**What to build:** A signed-in caller can ask for short-lived addresses to upload photos to a private bucket that is not served publicly. The call takes the number of photos wanted (up to the batch limit of 100) and returns, for each, a server-chosen key under the caller's own prefix, a presigned PUT address that fixes the content type to JPEG and expires shortly, and its expiry. The bucket is a new private R2 bucket with its own scoped credential, and its CORS lets the frontend origin PUT. `ObjectStore` gains a way to read an object's size, which ticket 04 needs. When the bucket is not configured the call refuses with a clear error and nothing else breaks, so local and preview environments work without it. The call is counted against the per-user rate limit.

Spec: `.scratch/scan-enhancements/spec.md` (Recording). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Pattern: the Admin image upload of `docs/adr/0019-catalog-content-is-curated-through-an-in-app-admin.md`.

**Blocked by:** None (can start immediately)

**Status:** done — implemented on `feat/scan-enhancements` (merge `ebdd2cb`), review fixes in `294165a`, not yet merged to `main`.

- [ ] Each returned key sits under the caller's own prefix and is unguessable
- [ ] The returned address accepts one PUT of a JPEG until it expires, and rejects another content type
- [ ] More than 100 photos in one request, and fewer than one, are refused
- [ ] With no bucket configured the call is refused with a clear, machine-readable reason and the rest of the API starts and works
- [ ] A Guest or an invalid token is rejected, and the call counts against the per-user rate limit
- [ ] The new bucket settings are validated at boot only when the bucket is configured, and are documented with the other R2 settings, including that this bucket must not be the public image bucket
- [ ] `ObjectStore` can report an object's size, with its mock regenerated
- [ ] The deployment notes name the new bucket, its credential and the CORS rule it needs
- [ ] The updated API contract is regenerated
- [ ] All of the above is covered by tests at the HTTP boundary against real Postgres, with the generated `ObjectStore` mock
