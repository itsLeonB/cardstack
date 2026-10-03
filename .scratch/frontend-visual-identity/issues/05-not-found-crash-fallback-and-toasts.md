# 05: Not-found page, crash fallback and toasts

**What to build:** Friendly handling for the states that need a standalone page, plus toast feedback for shell-level actions. A not-found page for unknown URLs and missing entities, a minimal root crash fallback, and toasts for logout. Spec: `.scratch/frontend-visual-identity/spec.md`.

**Blocked by:** 01

**Status:** done

- [x] Root not-found page rendered inside the shell, with actions back to Home and the Catalog
- [x] A missing Collection or Card resolves to the same not-found page with a message naming what was not found, rather than a generic message
- [x] Root crash fallback rendered inside the shell with a Reload action; neither page ever shows a stack trace or raw error text
- [x] Add the shadcn `sonner` toast and mount its toaster in the root. If existing ticket 31 has already landed and installed it, acknowledge that and reuse it instead
- [x] Logout shows a toast on failure (it is silent today) and still navigates to login on success. The session-expired notice and token refresh are owned by ticket 31, not this one
- [x] Existing inline errors (forms, dialogs, per-card quantity edits, page-load failures) are untouched; per-feature mutations are not migrated to toasts
- [x] Record the convention in the frontend conventions doc: toasts for failures of non-form actions; inline errors for form validation and for failed page data, with a Retry action
- [x] Playwright: an unknown URL shows the not-found page inside the shell; axe check on it
- [x] RTL (network mocked): crash fallback renders on a thrown error without exposing it; logout failure shows a toast
- [x] Lint, typecheck, tests and build pass; note anything not verified in a browser
