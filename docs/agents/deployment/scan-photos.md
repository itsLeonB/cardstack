# Scan photos

The photos of scan Corrections live in a private Cloudflare R2 bucket, separate from the public image bucket of `docs/agents/deployment/images.md` (ADR-0022). Nothing serves them publicly; an Admin reads them in the Admin app's review screen through short-lived presigned links (ADR-0027).

- API (Railway): `SCAN_PHOTO_R2_ACCOUNT_ID`, `SCAN_PHOTO_R2_ACCESS_KEY_ID`, `SCAN_PHOTO_R2_SECRET_ACCESS_KEY` and `SCAN_PHOTO_R2_BUCKET`, a secret on Railway. `SCAN_PHOTO_R2_ENDPOINT` is optional and replaces the address derived from the account id. Set all four or none: the API refuses to boot with a partial set, or when `SCAN_PHOTO_R2_BUCKET` equals `R2_BUCKET`. With none, `POST /scan/photo-uploads` and `POST /scan/feedback` answer 503 `photo_storage_not_configured` and the app records no feedback; scanning itself is unaffected. Local, CI and preview environments run this way. Local value: `backend/.env.example`.
- Bucket: a new R2 bucket with no public access, no custom domain and the `r2.dev` address off.
- Credential: an R2 API token scoped to that one bucket, with object read, write and delete: write for the presigned `PUT` and for keeping an approved cut, read for the size and first-bytes check when feedback is posted and for reading the cut an Admin approves, delete for removing an object that fails the feedback check. Never reuse the public image bucket's token.
- CORS (R2 dashboard, bucket settings), two rules, each with allowed header `Content-Type` and a short max age:
  - The user app's origins (`APP_CLIENT_URLS` in `docs/agents/deployment/clerk.md`), method `PUT`. Without it the browser's preflight fails and the client silently records nothing.
  - The Admin app's origins (`ADMIN_CLIENT_URLS` in the same file), methods `GET` and `PUT`. `GET` lets the Admin app fetch a stored photo: the review screen fetches a binder photo to cut its cell, and approval fetches every photo, card scans included, to upload it. Without it a binder cell shows "photo could not be loaded" and approval of any photo fails, while card-scan photos still display. `PUT` lets the Admin app upload the image it approves; without it approval fails before anything is embedded.

Keys, each signed for `image/jpeg` for five minutes where a presigned address exists:

- `<profile id>/<uuid>`: the user's photo, uploaded by the user app.
- `reviewed/<scan_feedback id>.jpg`: the Admin's upload for approval.
- `approved/<scan_feedback id>.jpg`: the exact bytes Approve embedded, written by the API only.

There is no expiry or deletion job. To remove a user's photos and what was learned from them, by hand: list their `scan_feedback` ids, delete the `<profile id>/` prefix and the `reviewed/<id>.jpg` and `approved/<id>.jpg` objects for each id, delete their `card_embeddings` rows whose `source` is `correction:<id>`, then delete their `scan_feedback` rows.
