# API protection: edge secret and rate limits

## Backend edge secret

- `APP_EDGE_SECRET` (Railway production): when set, the API rejects every request without a matching `X-Edge-Secret` header (403, except `/health`) and takes the client address from `CF-Connecting-IP`, ignoring `X-Forwarded-For` and `X-Real-IP`. Configure a Cloudflare Transform Rule on the API's proxied domain that adds `X-Edge-Secret: <value>` to every request.
- Leave it **unset in preview environments** (and locally): they are reached through their own hosting URLs, not Cloudflare, so a set secret would reject every request. When unset, no edge check runs and the client address comes from `X-Real-IP` as before.

## Backend per-user rate limits

Signed-in callers are limited per user (keyed by their verified user id) inside the API, so one account gets one allowance from any number of addresses. Each tier is a token bucket: a caller may spend the burst at once and gets the per-minute number back each minute. The name-searchable card lists (`GET /catalog/cards`, `GET /inventory/cards`, `GET /collections/{id}/entries`) share a tighter search tier, and the three facets routes (`GET /catalog/facets`, `GET /inventory/cards/facets`, `GET /collections/{id}/facets`) share a tighter facets tier; neither spends the general one. Over a limit, the API answers 429 with a `Retry-After` header in whole seconds and a fixed body. Guests are not limited per user: the Cloudflare edge rule and the in-API per-IP backstop (100 requests a second, burst 200) cover them. The API refuses to boot with any of these set to zero or less.

| Setting | Default |
| --- | --- |
| `RATE_LIMIT_USER_PER_MINUTE` / `RATE_LIMIT_USER_BURST` | 300 / 100 |
| `RATE_LIMIT_SEARCH_PER_MINUTE` / `RATE_LIMIT_SEARCH_BURST` | 60 / 30 |
| `RATE_LIMIT_FACETS_PER_MINUTE` / `RATE_LIMIT_FACETS_BURST` | 30 / 15 |

The buckets live in the API's memory: every deploy or restart resets them, and the limits only hold while the API runs as a single replica. Running more than one replica needs a shared store (for example Redis) before these limits mean anything.
