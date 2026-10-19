# 05: Tuning script

**What to build:** An offline script, built like the Phase 1 accuracy script, that the operator runs against a folder of photos to set the two provisional settings from data. For photos of binder Pages it cuts each cell the way the app does and prints the top similarity of every cell, grouped by photo, so the empty threshold can be set from an empty, a sparse and a full Page. Separately it prints the pairwise catalog embedding distances inside groups of Cards that share a normalized name, so the Reprint art distance can be set from real groups such as one artwork printed in several Expansion Sets. It is a report, not a CI test, and it changes nothing in the database.

Spec: `.scratch/scan-enhancements/spec.md` (Tuning). The operator runs it on their three Page photos; the agent builds it and shows it working on any photos available.

**Blocked by:** 01 (empty detection in the match response), 02 (reprint lookup route)

**Status:** ready-for-agent

- [ ] Given a folder of Page photos and a grid, it prints the top similarity of each cell, with the photo and the cell position
- [ ] It prints a summary per photo (lowest, highest, and how many cells fall below a threshold the operator passes in)
- [ ] It prints pairwise catalog distances within same-name groups, with the largest groups first and a way to limit the output
- [ ] It reuses the app's own cell cutting and name normalization so the numbers match what the API would compute
- [ ] It makes no writes, and fails with a clear message when the catalog has no embeddings
- [ ] Its usage and what each number means for each setting are documented next to the Phase 1 accuracy script
