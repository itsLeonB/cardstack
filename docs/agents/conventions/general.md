# General code conventions

These apply to every change, backend and frontend, whether made by the root agent or a subagent. Component-specific rules live in `backend.md` and `frontend.md` next to this file.

- Make the smallest change that satisfies the request: no cleanup around a bug fix, no abstractions for hypothetical needs, no handling for cases that can't happen.
- Comment only the non-obvious WHY: a hidden constraint, a workaround, a subtle invariant.
- Fix security issues when you spot them: injection, path traversal, XSS, secret leaks.
