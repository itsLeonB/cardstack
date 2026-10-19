# 12: Record card-scan feedback

**What to build:** When a Draft Addition is confirmed and the placement call succeeds, the client records what happened to every Scan, silently. It derives each Scan's kind from the Detection it holds and the final choice: the Card the matcher put first is a confirmation, whether or not the matcher was confident; another candidate is `switched`; one picked from "Other prints" is `reprint`; marking Empty or Unidentified when the Detection offered a Card is `empty` or `unidentified`; keeping what the Detection already was (Empty on an empty Detection, Unidentified on a Detection with no candidates) is a confirmation. Each row carries the Detection as received, the chosen Card, its rank and its source (`top` or `reprint`). Photos of Corrections are uploaded to the addresses the API returns, then the batch is posted. Removed Scans, failed matches left Unidentified, and anything unconfirmed record nothing. No failure of any of this is visible to the user or delays the confirm.

Spec: `.scratch/scan-enhancements/spec.md` (Recording, Frontend). Decision: `docs/adr/0022-scan-corrections-are-recorded-with-their-photo-in-a-private-bucket.md`. Vocabulary: Correction, Detection in `GLOSSARY.md`.

**Blocked by:** 04 (record scan feedback), 06 (Draft Addition keeps resolved Scans with their photos), 08 (empty and unidentified marks on card scan), 10 (other prints in the card-scan picker)

**Status:** ready-for-agent

- [ ] Confirming sends one row per resolved Scan with the right kind for each of the cases above
- [ ] The rank is the position in the list the matcher offered, or in the Reprints list counted across pages, and the source says which list
- [ ] A Correction's photo is uploaded and its key is in the row; a confirmation sends no key and uploads nothing
- [ ] A Correction whose photo is no longer available is sent with no key
- [ ] Removed Scans, a failed match left Unidentified, and a Draft Addition that is cleared or never confirmed send nothing
- [ ] The upload and feedback calls run after the placement call succeeds, and never delay or change its outcome
- [ ] A failed upload, a failed feedback call, and a refusal because the bucket is not configured are all silent and leave the confirmed result untouched
- [ ] The Draft Addition is cleared and its saved photos removed whether or not recording succeeded
- [ ] All of the above is covered by tests on the scan route with the generated API client mocked, including one test per kind
