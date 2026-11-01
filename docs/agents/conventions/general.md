# General code conventions

These apply to every change, backend and frontend, whether made by the orchestrator or a component agent. Component-specific rules live in `backend.md` and `frontend.md` next to this file.

- Make the smallest change that satisfies the request: no cleanup around a bug fix, no abstractions for hypothetical needs, no handling for cases that can't happen.
- Comment only the non-obvious WHY, and keep comments and docs minimal, brief and simple: the why and the how-to, with the code stating what it does. Name the concept rather than a file or path, which can move.
- Fix security issues when you spot them: injection, path traversal, XSS, secret leaks.
- Give each test layer its own job: unit tests own branches and edge cases of one unit; feature (backend) and end-to-end (frontend) tests own the functional correctness of one feature or of several that form one user flow, so they cover the happy path and the flow's key failure, not every branch again.
- Update every reference in the same change as the behaviour: a fix that removes behaviour (a guard, a retry, a code path) greps for its wording in handler summaries, `openapi.json` and `docs/` and updates each hit, as it does for anything `docs/agents/conventions/*.md` or `GLOSSARY.md` describes.
- Give every production seam a production caller: a helper, export or injected dependency that only tests call moves into the test or into a module both sides import. In Go, test-only helpers stay in `_test.go`.

## Convention docs

- Enforce with tooling first. If a linter, formatter or type checker can catch a rule, configure that tool instead of writing the rule here: `code-review` skips whatever tooling enforces.
- Add a bullet to `docs/agents/conventions/*.md` only for a rule that holds across modules and that a reviewer can check in a diff. Put how one module works in that file's header or doc comment, a decision with rejected alternatives in an ADR (`docs/adr/`), and ticket-specific detail in the ticket.
- Keep each bullet under 900 characters; `scripts/check-conventions.sh` fails the Conventions CI job on a longer one, because a bullet that long is explaining a module.
- Review a diff that adds or lengthens a convention bullet against the first rule, and ask for the module-level explanation to move out.
