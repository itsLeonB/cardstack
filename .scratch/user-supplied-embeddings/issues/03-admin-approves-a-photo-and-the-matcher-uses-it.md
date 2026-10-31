# 03: Admin approves a photo, and the matcher uses it

**What to build:** On the review screen the Admin approves a photo. The Admin app uploads the image to embed (a binder cell's cut, or a card-scan photo as stored) to the private bucket, then approves. The photo is embedded as an extra image of the Card the user chose, and the next match of the same photo scores that Card higher. Nothing is half-approved if the embedding fails.

Spec: `.scratch/user-supplied-embeddings/spec.md` (Embedding, Photo access and approval, Admin app). ADR-0027, ADR-0022. Vocabulary: User-supplied Embedding in `GLOSSARY.md`.

**Blocked by:** 01, 02

**Status:** ready-for-agent

- [ ] The API mints a presigned PUT for a server-chosen key under a reviewed prefix of that item, with a signed content type and a short expiry, for an Admin only
- [ ] Approve names that key; the API refuses a key outside that item's prefix, a non-JPEG first bytes and an object over the size cap
- [ ] Approve embeds the object synchronously with the embedder the matcher uses for query photos, then in one transaction inserts the embedding and sets the status and reviewer
- [ ] The embedding is for the configured model, for the Card the user chose only (not its Reprints), and its source is `correction:` followed by the feedback row's id; the unique key on Card, model and source is unchanged
- [ ] If the embedding service fails, nothing is stored, the item stays pending and the Admin can retry
- [ ] Approving an item that is already reviewed answers conflict and leaves one embedding
- [ ] An item whose chosen Card has since been deleted cannot be approved
- [ ] The Admin app uploads, then approves, in that order, and a failed approval keeps the item in the queue; an item taken by another Admin shows the conflict
- [ ] Approve removes the item from the queue, and the cut object stays in the bucket
- [ ] Route tests against real Postgres with a faked embedder and object store cover the above, and show an approved photo ranks its Card higher for the same photo
