# 12: Limit search for card scanning

**What to build:** when scanning cards into a Collection (a Draft Addition), the user can limit the search to chosen Expansion Sets and/or rarities from a "Limit search" row above the camera, and the matcher then offers only Cards inside that Scan Filter, other prints included (GLOSSARY.md: Scan Filter).

**Blocked by:** None (can start immediately).

**Status:** done

- [x] The scan match and reprint lookups accept optional Expansion Set and rarity restrictions, so the nearest-Card search and the other-prints suggestions both stay inside them. Empty means no restriction, and unpublished sets stay excluded as before.
- [x] The matcher never widens a restricted search by itself. Before the restriction is added, the existing confidence rule is checked on a narrowed pool so that restricting it cannot make a poor match look confident; the absolute-score bar still holds.
- [x] The chip row reads "All sets · All rarities" by default and opens a sheet with two multi-select groups and Clear. Selected values show as wrapping chips, and the sheet applies on close. A change mid-session affects only photos taken afterwards.
- [x] The last Scan Filter is remembered on the device in local storage and is not stored by the backend.
- [x] A photo that matches nothing inside the filter is marked Unidentified as today with a note such as "Searched MA5, MA6 only". There is no search-all button.
- [x] Corrections and scan feedback are recorded as before for a limited scan.
- [x] HTTP-boundary tests cover the restrictions on match and reprints and the poor-match case. Screen tests cover the row, the sheet, the chips, the remembered choice and the note.
