# 08: Production cutover

**What to build:** The operator moves the Admin to the Admin app in production, following the runbook from ticket 07: chooses the hostname on a subdomain of the production domain, adds it to Clerk, creates the Vercel project with the production Clerk key and API address, adds the origin to R2 CORS, sets `ADMIN_CLIENT_URLS` in Railway before the merge, then deploys the API, the Admin app and the user app without `/admin` in that order. The operator checks every screen with a real set, and keeps rollback (revert the release) ready.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** 07

**Status:** ready-for-human

- [ ] Hostname chosen and added in Clerk, Vercel and R2 CORS, and `ADMIN_CLIENT_URLS` set in Railway before the merge
- [ ] API, Admin app and user app deployed in the documented order, and the Admin can sign in to the Admin app without a second login
- [ ] An Expansion Set can be edited and a Card added with an image from the Admin app; unpublished sets show there and not in the user app
- [ ] The user app's `/admin` answers the not-found page, and the Admin app is not indexed
- [ ] The cutover is recorded here: date, hostname (not in the repository docs), anything that surprised
