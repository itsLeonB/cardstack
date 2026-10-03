# Card and Expansion Set images are rehosted on R2 and transformed at read time

The ingester used to store the third-party image address verbatim, and every tile hot-linked a full-size PNG from that host. We now copy each original, unmodified, into our own R2 bucket during ingestion and serve it from a Cloudflare-proxied subdomain, where Cloudflare Image Transformations resize it and pick a modern format at read time. Keys are deterministic from the row identifier, so re-running the ingester is the backfill and skips what is already hosted. The row stores a hosted-image key and keeps the scraped source address in its own column; the API builds its image address from a configured base address plus the key and returns an empty address when there is no key.

A failed download or upload leaves the key empty and the frontend shows its existing name placeholder. We do not fall back to the source address, because the point is to stop hot-linking and a silent fallback would keep it alive forever. The ingester only fetches from the known source host, enforces a size limit and checks the content is an image, since the address comes from scraped markup.

## Considered Options

- Pre-generate fixed sizes at ingestion and store each as its own R2 object. Rejected: more ingestion code and storage, and it freezes the sizes the UI can ask for, while the free transformation allowance covers a personal-scale app.
- Proxy images through our own API. Rejected: it spends API capacity on bytes that a CDN serves better.

## Consequences

Cloudflare's free plan allows 5,000 unique transformations per month. Beyond that new variants return an error rather than being billed, so the frontend requests a redirect to the original as the fallback. Only images actually viewed count, and the guest preview page cap limits how fast a scraper can use the allowance. Transformations need a domain on Cloudflare, which Clerk's production instance needs anyway.
