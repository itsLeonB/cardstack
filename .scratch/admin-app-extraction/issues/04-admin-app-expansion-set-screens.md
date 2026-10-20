# 04: Admin app: Expansion Set screens

**What to build:** An Admin creates and edits an Expansion Set in the Admin app exactly as in the old `/admin`: a form to create a set (409 on a duplicate code), an edit form that loads the existing set and sends every field back (PUT replaces), the code not editable, and the Admin seeing unpublished sets. The screens, the shared form helpers and their tests move over from the user app with behaviour unchanged. The user app's copy stays until ticket 06.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** 03

**Status:** resolved

- [ ] An Admin can create an Expansion Set and gets a clear message on a duplicate code
- [ ] An Admin can open an existing Expansion Set (published or not), edit it, and the request carries every field
- [ ] The code of a set cannot be edited, and rarity defaults only on create, as ADR-0019 says
- [ ] The Expansion Set form tests and the shared form helper tests moved over and pass with behaviour unchanged
- [ ] The Admin app's typecheck, lint, tests and build pass
