# 02: Cloudflare, R2 and hosting setup

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The owner's domain on Cloudflare with an R2 bucket served from a `cdn` subdomain, image transformations enabled, the API served from a proxied `api` subdomain protected by a secret header and a coarse per-IP rule, and all resulting variables set on Railway, Vercel and GitHub. Only the owner can do this; deliver it as an interactive bash wizard (the `wizard` skill).

**Blocked by:** None (can start immediately).

**Status:** done — wizard delivered in `scripts/cloudflare-setup.sh`; acceptance boxes stay unticked until the owner has run it against the real Cloudflare account.

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

**Wizard:** `scripts/cloudflare-setup.sh` (9 stages, re-runnable; progress in `~/.local/state/cardstack/cloudflare-wizard.state`, values in the mode-600 `~/.local/state/cardstack/cloudflare.env`, never in the repo). It verifies the R2 key with a signed upload, the public image host and a transformed variant with `curl`, the proxied API with `/health` (needs a `cf-ray` header), and the rate rule with a burst that must return Cloudflare's 429 (error 1015). It cannot check the secret-header rule, since nothing rejects requests until ticket 04.

**Names tickets 04, 05 and 06 should adopt:** backend (Railway) `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `IMAGE_BASE_URL` (`https://cdn.<domain>`, no trailing slash) and `EDGE_SECRET`; frontend (Vercel, Production and Preview) `VITE_IMAGE_HOST` (`https://cdn.<domain>`). The edge secret travels in the request header `X-Edge-Secret`, set by a Transform Rule scoped to `api.<domain>`; the client address comes from `CF-Connecting-IP`. The transformation path is `https://cdn.<domain>/cdn-cgi/image/width=240,format=auto,onerror=redirect/<key>`. No GitHub secrets are needed: nothing in CI reads these yet.

**Findings that change later tickets:**

- The free plan's rate rule can only match on path (no hostname), so it is zone-wide. The wizard scopes it to `/catalog`, `/collections` and `/inventory` so image loads on `cdn.` never count; the frontend's own domain should stay "DNS only" in Cloudflare or its `/catalog` pages would count too. The default is 100 requests per 10 s per IP.
- Railway needs both a CNAME and a TXT record for the custom domain, and Cloudflare SSL mode "Full" (not "Full (strict)"). Railway's docs say proxying works only for a first-level subdomain such as `api.<domain>`.
- Railway pull-request environments are forked from the base environment and may inherit `EDGE_SECRET` and the R2 keys. Ticket 04 must make the preview workflow unset `EDGE_SECRET` in the PR environment (acceptance: previews have no edge secret), and ticket 05 should do the same for the R2 keys.
- The wizard asks before putting `EDGE_SECRET` on Railway, because once ticket 04 ships it breaks a frontend that still calls the raw Railway address. Ticket 13 step 5 sets it at cutover if it was withheld.
