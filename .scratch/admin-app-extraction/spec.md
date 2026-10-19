# Admin app extraction

Status: needs-triage

Draft: not grilled. Every point below comes from one conversation and is an idea or a known fact, not a settled decision. Run a grilling session before turning this into a spec or tickets.

## Why this exists

The Admin (GLOSSARY.md) writes catalog content through an in-app `/admin` area of the user app (ADR-0019). The scan enhancements spec (`.scratch/scan-enhancements/spec.md`) adds recorded Corrections and photos, and the user-supplied embeddings spec (`.scratch/user-supplied-embeddings/spec.md`) adds an Admin review of user photos before they are embedded. That grows the Admin's job. The operator's view is that the Admin should be its own app, separate from the user app, rather than a growing corner of it.

## Known facts

- ADR-0019 puts the Admin inside the user app: the allowlist is `ADMIN_EMAILS`, matched against the Clerk token's verified `email` claim, and an Admin is an ordinary Auth Identity that happens to be listed, with no role stored in Cardstack.
- Admin image uploads use a presigned PUT to R2 with a server-chosen key, a signed content type and a signature check on confirm. The file checks live in the admin frontend.
- A separate admin app would supersede part of ADR-0019 and probably needs its own ADR.
- This is the second of three specs, in the order: scan enhancements, admin app extraction, user-supplied embeddings.

## Questions the grilling must settle

- What moves out: only the screens, or also the backend routes and the allowlist? One API with two frontends, or a second API?
- Where does it deploy, on what domain, and how is it kept from being reachable or indexed by users?
- Does it keep Clerk and the `ADMIN_EMAILS` allowlist, or does the separation change how an Admin is identified?
- What does extraction do to the existing catalog-admin screens, their tests and their tickets, and how is the move done without a gap in the Admin's access?
- Which of the Admin's new responsibilities (reading Corrections, reviewing user photos) are in the first version of the new app?
- Is the user app's `/admin` removed at once or kept until the new app is live?
- Shared code between the two frontends: the generated API client, the upload helper, the design tokens. Duplicate, share a package, or a monorepo workspace?

## Out of scope for now

- The review of user-supplied photos themselves (that is the user-supplied embeddings spec).
- Anything about the user app's scanning.
