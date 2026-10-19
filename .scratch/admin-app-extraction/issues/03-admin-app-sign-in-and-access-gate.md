# 03: Admin app: sign-in and access gate

**What to build:** A new top-level `admin/` package that an Admin can run locally on its own port against the local API and the development Clerk instance. It is built like the user app (TanStack Start in SPA mode, Vite, Tailwind 4, orval, bun, oxlint, vitest) with its own copy of the shadcn components and design tokens it needs. A Guest, or a signed-in user who is not listed, sees the ordinary not-found page; an Admin sees an empty shell with sign-out. A failed access check (429, 5xx) shows an error, not not-found. It serves `noindex`, a `robots.txt` that disallows everything and no sitemap, and its production build fails when the Clerk publishable key or the API address is missing. It has its own orval target reading the same `openapi.json`, limited to the admin operations and the catalog operations it will call. A CI job runs typecheck, lint, tests and build on changes to it and is a required check.

Spec: `.scratch/admin-app-extraction/spec.md`. Decision: `docs/adr/0024-the-admin-is-its-own-app-and-its-routes-accept-only-its-origin.md`. Vocabulary: Admin app, Admin, Published in `GLOSSARY.md`. Conventions: `docs/agents/conventions/general.md` and `frontend.md`.

**Blocked by:** 01, 02

**Status:** ready-for-agent

- [ ] `bun run dev` in the package serves the app on its own port; with the local API and a listed email an Admin signs in and sees the shell
- [ ] A Guest and an unlisted signed-in user see the same not-found page, and a 429 or 5xx from the access check shows an error
- [ ] The access check asks the API and treats 204 as an Admin and 404 as not found, as the old `/admin` gate did; a Guest is not even asked
- [ ] Responses carry `noindex`, `robots.txt` disallows everything, and no sitemap is built
- [ ] A production build exits non-zero when the Clerk key or the API address is unset, and warns and continues elsewhere
- [ ] The generated client is produced by `bun run codegen` from the same `openapi.json` and holds only the operations the Admin app calls
- [ ] Component tests of the gate (Admin, Guest, unlisted, failed check) run with the generated client mocked
- [ ] A CI workflow for the package runs typecheck, lint, tests and build, and is listed as a required check
- [ ] The package's local settings example lists the Clerk key and the API address, and the local Admin origin is the one in the API's `ADMIN_CLIENT_URLS` example
