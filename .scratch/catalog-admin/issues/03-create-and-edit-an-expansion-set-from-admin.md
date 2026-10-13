# 03: Create and edit an Expansion Set from admin

**What to build:** An Admin can create an Expansion Set (name, code, release date, and a Series chosen from the existing ones) or edit an existing one, from the `/admin` page. Saving marks the set Curated. Creating `SM-P` ("Kartu Promo Matahari & Bulan", release date 2019-05-08, no cover, under "Matahari & Bulan") makes it appear in catalog browse like any other set.

**Blocked by:** 01, 02.

**Status:** resolved

- [ ] An admin operation upserts an Expansion Set by code, rejecting an unknown Series with a validation error, and marks the row Curated
- [ ] The set form on `/admin` creates a new set and edits an existing one, including one that came from the Asia site
- [ ] Adding a set with an older release date does not unexpectedly reorder Series (Series order is derived from the earliest set date; `SM-P` stays within the already-oldest Series)
- [ ] The new set appears in catalog browse under its Series, and a set with no cover renders text-only as other cover-less sets do
- [ ] Route tests on real Postgres cover create, edit, unknown Series, and the Curated flag; a frontend test covers the form
