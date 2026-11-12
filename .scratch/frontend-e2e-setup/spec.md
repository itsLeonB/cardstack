# Local authenticated frontend E2E reliability

Make authenticated frontend E2E tests reliable in a fresh worktree using the supported Compose stack and real Clerk test credentials.

## Issues

1. [Prepare the backend environment for frontend E2E](issues/01-prepare-backend-env-for-frontend-e2e.md).
2. [Make the local sign-in browser test reliable](issues/02-investigate-local-sign-in-timeout.md).

The sign-in timeout has no confirmed cause. Do not assume that fixing the backend environment will fix it. Preserve guest-only and CI runs, real Clerk coverage, and secret safety.
