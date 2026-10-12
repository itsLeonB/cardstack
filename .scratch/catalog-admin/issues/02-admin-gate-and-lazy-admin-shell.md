# 02: Admin gate and lazy `/admin` shell

**What to build:** The access control and an empty admin page, end to end. The operator names Admins in an env setting; only a signed-in caller whose verified Clerk email is on that list reaches admin routes, everyone else gets a 404, and the frontend `/admin` page is a code-split route linked from nowhere. When the allowlist or the R2 write settings are missing, the admin routes do not exist at all (ADR-0019, glossary entry Admin).

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A comma-separated allowlist setting is parsed at boot; an empty entry or a value that is not an email stops the API with an error naming the entry, in the style of the client-origin setting
- [ ] The R2 write settings (account, key, secret, bucket) are read by the API; the admin route group is registered only when both the allowlist and the R2 settings are present
- [ ] A listed caller (compared case-insensitively against the token's `email` claim) passes; a signed-in unlisted caller and a guest both get a 404 from every admin route
- [ ] Admin traffic goes through the same edge secret and rate limits as the rest of the API
- [ ] A route-split `/admin` page loads only when opened by URL, has no navigation, sitemap or indexing entry, and shows the ordinary not-found page to a non-Admin
- [ ] The admin code is not in the main bundle (verifiable from the build output)
- [ ] Route tests on real Postgres cover listed, unlisted, guest, routes-absent-when-unset, and malformed-allowlist boot failure; a frontend route test covers the non-Admin not-found state
- [ ] The deployment docs describe the allowlist setting and state that the API now needs R2 write credentials (fix the image-hosting doc's claim that only the ingester has them); note that Clerk must keep requiring a verified email at sign-up
