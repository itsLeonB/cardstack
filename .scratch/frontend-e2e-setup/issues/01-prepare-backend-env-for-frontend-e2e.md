# 01: Prepare the backend environment for frontend E2E

**What to build:** A developer can run frontend E2E tests from a fresh worktree without authenticated API calls failing because the backend uses placeholder Clerk settings.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A fresh frontend worktree obtains the backend environment needed for authenticated E2E tests through the supported setup, without exposing secret values.
- [ ] The setup or verification step reports missing required backend settings before tests fail with misleading API errors, while preserving supported guest-only and CI runs.
- [ ] Automated checks cover a fresh worktree with available settings and the missing-settings case.
