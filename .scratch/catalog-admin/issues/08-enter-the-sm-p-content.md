# 08: Enter the SM-P content

**What to build:** The owner enters the `SM-P` set and its cards through the admin and makes them scannable. Only cards the owner can photograph exist, because a Card requires an image; the rest stay missing until they are photographed.

**Blocked by:** 03, 05, 07.

**Status:** ready-for-human

- [ ] The `SM-P` Expansion Set exists, named "Kartu Promo Matahari & Bulan" under "Matahari & Bulan", release date 2019-05-08, no cover
- [ ] Each card the owner holds is entered from the checklist with a flat, cropped, front-on photo; numbers not yet photographed are listed in the checklist's header
- [ ] Names are corrected from the physical card where they differ from the English list
- [ ] The entered cards appear in browse and search under the set and can be added to a Collection
- [ ] `make embed-catalog ARGS="-set SM-P"` is run, then `make collect-embeddings` once the batch finishes, and a real photo of an entered card matches the right Card on the deployed API
- [ ] Clerk is confirmed to still require a verified email at sign-up before the admin is used on production
