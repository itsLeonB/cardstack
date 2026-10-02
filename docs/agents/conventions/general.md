# General code conventions

These apply to every change, backend and frontend, whether made by the root agent or a subagent. Component-specific rules live in `backend.md` and `frontend.md` next to this file.

- Make the smallest change that satisfies the request: no cleanup around a bug fix, no abstractions for hypothetical needs, no handling for cases that can't happen.
- Comment only the non-obvious WHY: a hidden constraint, a workaround, a subtle invariant.
- Fix security issues when you spot them: injection, path traversal, XSS, secret leaks.
- Give each test layer its own job: unit tests own branches and edge cases of one unit; feature (backend) and end-to-end (frontend) tests own the functional correctness of one feature or of several that form one user flow, so they cover the happy path and the flow's key failure, not every branch again.
- Update the doc in the same change when it alters behavior that `docs/agents/conventions/*.md` or `CONTEXT.md` describes, so those docs never go stale.
- Keep code that only tests call (helpers, seams) in `_test.go` files, never in production packages.
