// Which environment variables switch the signed-in end-to-end specs on. Kept
// free of Playwright imports so a unit test can cover it (see
// docs/agents/testing.md, "End-to-end authentication").
export interface ClerkE2eCredentials {
  email: string
  password: string
}

export const CLERK_E2E_SKIP_REASON =
  "Clerk sign-in e2e is not configured: set CLERK_SECRET_KEY, E2E_CLERK_USER_EMAIL and E2E_CLERK_USER_PASSWORD (fork pull requests and local runs without them skip these specs)."

// All three must be present: the secret key mints the Testing Token and the
// sign-in token, the other two are the dedicated test user. A partly
// configured run skips rather than failing half way through.
export function readClerkE2eCredentials(
  env: Record<string, string | undefined>
): ClerkE2eCredentials | null {
  const secretKey = env.CLERK_SECRET_KEY?.trim()
  const email = env.E2E_CLERK_USER_EMAIL?.trim()
  const password = env.E2E_CLERK_USER_PASSWORD
  if (!secretKey || !email || !password) return null
  return { email, password }
}
