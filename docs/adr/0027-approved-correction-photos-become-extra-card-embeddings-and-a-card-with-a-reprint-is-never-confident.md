# Approved Correction photos become extra Card Embeddings, and a Card with a Reprint is never confident

A user's photo of a Card the matcher got wrong is a real photograph of that print, and the matcher can already hold several embeddings per Card (ADR-0018). So an Admin reviews the photos kept by ADR-0022, and each one approved is embedded as an extra image of the Card the user chose. Nothing the user uploads reaches the matcher without that approval.

Corrections of kind `switched`, `reprint` and `placed` that have an image are eligible, old ones included, with no automatic quality gate. The Admin app shows the photo (a binder cell is cut from the stored Page photo with the grid and corners, using the user app's `CELL_MARGIN`) next to the Card's catalog image and the stored scores, and offers approve or reject. A rejection is remembered. Review state is three columns on `scan_feedback` (`review_status`, `reviewed_by`, `reviewed_at`), NULL meaning pending. The Admin app uploads its cut by presigned PUT to `reviewed/<feedback id>.jpg` in the private bucket and names that key on approve; the API checks it, embeds it synchronously and inserts the row in one transaction with the status. The Admin reads photos through short-lived presigned GETs, never a bucket credential. A user-supplied embedding is only ever an embedding, never a catalog image.

The row's `source` is `correction:<scan_feedback id>`. The source already names the image that was embedded, and one source per Correction keeps `UNIQUE (card_id, model, source)` as it is, so no migration loosens it. Queries about the catalog itself (the Reprint art distance of ADR-0023 and the tuning script) must filter `source = 'catalog'`, or a user photo would count as the Card's art. The matcher uses all sources at once, with no flag and no cap, and its thresholds and margin do not change.

User photos can make one print of a family score above its Reprints, which would turn a near-tie, the only thing that keeps a Reprint result non-confident today, into a confident wrong print. A result is therefore never confident when its top Card has a Reprint, whatever the margin says. This keeps today's behaviour and costs one Reprint lookup, made only when the result would otherwise be confident.

## Considered Options

- A `MATCH_USER_EMBEDDINGS` flag, off until measured, with a catalog-only mode in the tuning script for a before-and-after. Rejected by the operator: embeddings are on by default and the gain is not measured first.
- A separate `correction_reviews` table. Rejected by the operator in favour of columns on `scan_feedback`.
- A cap on user embeddings per Card, and a Remove action in the Admin app. Both deferred until the data shows a need.

## Consequences

`source` is a string, not a foreign key, so deleting a `scan_feedback` row or a user's data leaves their embeddings behind. The operator accepts this gap now, as ADR-0022 accepts manual deletion; account deletion, when it exists, must delete the embeddings of the user's Corrections too. With no flag, no cap and no Remove action, the only way back is SQL (`DELETE FROM card_embeddings WHERE source LIKE 'correction:%'`). A change of `EMBEDDING_MODEL` leaves user-supplied rows unused, because the matcher compares one model; the stored cut lets a later command re-embed them, and none exists yet. Photos, rejected ones included, stay until the operator deletes them by hand, and the scan screens gain a notice that photos of corrected scans may be used to improve matching, with no opt-out.
