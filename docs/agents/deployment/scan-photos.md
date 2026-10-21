# Scan photos

The photos of scan Corrections live in a private Cloudflare R2 bucket, separate from the public image bucket of `docs/agents/deployment/images.md` (ADR-0022). Nothing serves them publicly; the operator reads them by hand.

- API (Railway): `SCAN_PHOTO_R2_ACCOUNT_ID`, `SCAN_PHOTO_R2_ACCESS_KEY_ID`, `SCAN_PHOTO_R2_SECRET_ACCESS_KEY` and `SCAN_PHOTO_R2_BUCKET`, a secret on Railway. `SCAN_PHOTO_R2_ENDPOINT` is optional and replaces the address derived from the account id. Set all four or none: the API refuses to boot with a partial set, or when `SCAN_PHOTO_R2_BUCKET` equals `R2_BUCKET`. With none, `POST /scan/photo-uploads` answers 503 `photo_storage_not_configured` and the app records no feedback; scanning itself is unaffected. Local, CI and preview environments run this way. Local value: `backend/.env.example`.
- Bucket: a new R2 bucket with no public access, no custom domain and the `r2.dev` address off.
- Credential: an R2 API token scoped to that one bucket, with object read, write and delete: write for the presigned `PUT`, read for the size and first-bytes check when feedback is posted, delete for removing an object that fails it. Never reuse the public image bucket's token.
- CORS (R2 dashboard, bucket settings): allowed origins = the user app's origins (`APP_CLIENT_URLS` in `docs/agents/deployment/clerk.md`), method `PUT`, allowed header `Content-Type`, a short max age. Without it the browser's preflight fails and the client silently records nothing.

Keys are `<profile id>/<uuid>`, signed for `image/jpeg` for five minutes. There is no expiry or deletion job: to remove a user's photos, delete the `<profile id>/` prefix in the bucket and their rows in the database by hand.
