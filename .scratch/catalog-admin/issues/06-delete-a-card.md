# 06: Delete a Card

**What to build:** An Admin can delete a Card they entered by mistake. Deleting removes the Card row, its hosted image object and its embeddings, so nothing orphaned is left behind.

**Blocked by:** 04.

**Status:** resolved

- [ ] An admin delete operation removes the Card, its hosted image object from R2 and its embeddings
- [ ] Deleting a Card that is in an outstanding embedding batch is handled the way the embedding commands already handle a missing Card
- [ ] Inventory Entries that reference the deleted Card are handled deliberately: the behaviour is decided and covered by a test rather than left to a database error
- [ ] A listed Admin can delete; an unlisted caller and a guest get a 404
- [ ] Route tests on real Postgres with the object-store mock cover the deletion of row, object and embeddings; the delete control is on the `/admin` page and covered by a frontend test
