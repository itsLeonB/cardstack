# 13: Production rollout

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The hardened system is live: production Clerk on the owner's domain, the new schema deployed, every image hosted on R2 and served through the image host, and the API reachable only through Cloudflare. Done by the owner with this checklist, in order.

**Blocked by:** 01 (production half), 02, 04, 05, 06, 07, 08, 09, 10, 11, 12.

**Status:** done — rollout confirmed by the owner (2026-10-05), checklist and acceptance criteria included.

## Checklist

1. Confirm the production Clerk instance exists on the owner's domain with the same settings as development, and its keys are set on Railway and Vercel.
2. Accept the data wipe once more: the schema migration deletes all existing users, profiles, collections and Inventory Entries on the production database. Take a Neon branch or snapshot first if anything might be wanted back.
3. Deploy the backend, then the frontend. Confirm the migration ran.
4. Run the full ingester against production to backfill hosted images (no flags). It is long-running; re-run until no image failures remain.
5. Turn on the edge secret on Railway only after the `api` subdomain is live and the frontend's API base points at it. Confirm the raw Railway address now rejects requests.
6. Smoke test the live site: sign up with Google, sign up with email and verify, reset a password, browse as a Guest (one page, locked filters), sign in and use filters, create a Collection, add cards, check images load through the image host with the right sizes.

## Acceptance criteria

- [x] Every checklist step is done and the smoke test passes.
- [x] No catalog row still points at the source host for its image, or the remaining failures are recorded here.
- [x] The raw hosting address rejects requests without the secret header; the health check still works.
- [x] The deployment doc describes the final topology (domains, edge, image host, Clerk instances).

## Comments
