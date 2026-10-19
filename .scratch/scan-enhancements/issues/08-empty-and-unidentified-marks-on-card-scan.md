# 08: Empty and Unidentified marks on single card scan

**What to build:** Single card scan gets the two marks a binder cell already has, worded for a photo. Empty means no card was in the photo, and Unidentified means it is a card but none of the candidates fits. When the matcher returns an empty Detection, the Scan shows "no card found" with Retry and Remove and the marks. A marked Scan stays in the list, adds nothing to the Collection, and can be changed back to a Card before confirming. When the Draft Addition is confirmed, marked Scans are set aside and the summary says how many were set aside. This applies to the existing candidate picker; the binder's pill and menu keep their layout. A mark persists across a refresh with the Scan.

Spec: `.scratch/scan-enhancements/spec.md` (Frontend). Vocabulary: Scan, Unidentified, Detection in `GLOSSARY.md`.

**Blocked by:** 01 (empty detection in the match response), 06 (Draft Addition keeps resolved Scans with their photos)

**Status:** ready-for-agent

- [ ] A Scan whose Detection is empty shows "no card found", offers Retry and Remove, and has no candidate to pick
- [ ] Any resolved Scan can be marked Empty or Unidentified from the picker, and changed back to a candidate
- [ ] The wording of both marks is about the photo, not about a Slot
- [ ] A marked Scan is not counted into the addition and does not appear in the quantities written on confirm
- [ ] The confirm summary names how many Scans were set aside, and nothing is set aside silently
- [ ] A marked Scan keeps its mark across a refresh
- [ ] A Draft Addition holding only marked Scans can be confirmed or cleared without writing quantities
- [ ] Existing behaviour for unmarked Scans is unchanged
- [ ] All of the above is covered by tests on the scan route with the generated API client mocked
