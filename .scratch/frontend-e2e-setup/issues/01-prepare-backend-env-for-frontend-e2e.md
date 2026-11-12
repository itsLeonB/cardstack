# 01: Prepare the backend environment for frontend E2E

**What to build:** One worktree bootstrap prepares the backend, frontend, and Admin app together, so a developer can run frontend E2E tests from a fresh worktree without authenticated API calls failing because the backend uses placeholder Clerk settings.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] One supported bootstrap command replaces the backend and frontend bootstrap scripts, copies available `.envrc`, `backend/.env`, `frontend/.env`, and `admin/.env` files from the main checkout, starts backend tooling and the database, and prepares frontend and Admin app tooling and dependencies.
- [x] Bootstrap never overwrites an existing worktree env file, reports skipped files, and does not expose secret values.
- [x] Signed-in E2E verification reports missing or placeholder required backend Clerk settings before tests run, without exposing their values; guest-only and CI runs remain supported.
- [x] Automated checks cover fresh worktrees with settings available and missing, repeated bootstrap without overwriting existing files, and signed-in and guest-only verification.
- [x] Update callers, agent role instructions, and documentation to use the new command; remove the old bootstrap scripts.
