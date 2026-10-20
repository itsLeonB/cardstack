# 06: Remove `/admin` from the user app

**What to build:** The user app has no admin code. `/admin` answers the ordinary not-found page like any unknown address, and the route, components, tests, upload helper and form helpers are deleted. The one user-app file that imported a style constant from the admin form helpers gets its own copy of it. The user app's orval target leaves the admin operations out, so none are generated into its bundle. The route tree is regenerated.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`.

**Blocked by:** 04, 05

**Status:** resolved

- [ ] `/admin` in the user app answers the not-found page, tested
- [ ] No file under the user app imports an admin component, helper or generated admin operation
- [ ] The binder layout form looks and works as before, its existing tests pass
- [ ] `bun run codegen` in the user app produces a client without the admin operations, and the generated files are regenerated, not hand-edited
- [ ] The user app's typecheck, lint, tests, build and the existing Playwright suite pass
