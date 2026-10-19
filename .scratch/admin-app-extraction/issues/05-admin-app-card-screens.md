# 05: Admin app: Card screens

**What to build:** An Admin adds, edits and deletes Cards with images in the Admin app exactly as in the old `/admin`: find a Card by Expansion Set and number, the image checks and preview before saving, the presigned upload straight to R2 and the confirm step with the signature check, fast card entry (save and next), and delete behind a confirmation that names what goes. The Card form, the delete form, the upload helper and their tests move over from the user app with behaviour unchanged. The user app's copy stays until ticket 06.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] An Admin can add a Card with an image, and a failed upload or confirm says what failed
- [ ] An Admin can find an existing Card by set and number, edit it with every field sent back, and delete it only after a confirmation that names it; nothing is sent before it
- [ ] Fast entry (save and next) works as before
- [ ] A PUT from the Admin app's origin to the image bucket works when the bucket's CORS allows it (documented in ticket 07, performed in ticket 08)
- [ ] The Card form, delete form and upload helper tests moved over and pass with behaviour unchanged
- [ ] The Admin app's typecheck, lint, tests and build pass
