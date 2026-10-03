# 02: Cloudflare, R2 and hosting setup

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The owner's domain on Cloudflare with an R2 bucket served from a `cdn` subdomain, image transformations enabled, the API served from a proxied `api` subdomain protected by a secret header and a coarse per-IP rule, and all resulting variables set on Railway, Vercel and GitHub. Only the owner can do this; deliver it as an interactive bash wizard (the `wizard` skill).

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

## Steps the wizard covers

- Move the domain's DNS to Cloudflare if it is not already there.
- Create the R2 bucket, attach the `cdn` subdomain as its public custom domain, and create an R2 access key limited to that bucket.
- Enable image transformations on the zone.
- Add the `api` subdomain pointing at the Railway service, proxied, with SSL mode "Full". Confirm Railway's custom domain works behind the proxy (an assumption from the spec that was not verified).
- Add a Transform Rule that sets the secret header on requests to the API, and generate the secret.
- Add the single free-plan per-IP rate rule on the API subdomain.
- Set the new backend variables on Railway (R2 credentials, bucket, image base address, the edge secret) and the frontend image host on Vercel. Leave the edge secret unset in preview environments.

## Acceptance criteria

- [ ] An object uploaded to the bucket is readable at the `cdn` subdomain, and a transformed variant of it loads through the transformation path.
- [ ] A request to the API through the `api` subdomain succeeds; the same request to the raw Railway address without the secret header is later rejected once ticket 04 ships.
- [ ] The per-IP rule triggers when exceeded (tested with a burst).
- [ ] All variables are set on the right platforms, none committed, and preview environments are left without the edge secret.
- [ ] The wizard script is re-runnable and skips steps that are already done.

## Comments
