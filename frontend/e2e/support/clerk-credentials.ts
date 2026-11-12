// Decides whether the signed-in end-to-end specs run, skip or fail, from the
// environment alone. No Playwright imports, so a unit test covers it.
export interface ClerkE2eCredentials {
  email: string
  password: string
}

type Env = Record<string, string | undefined>

// All three are needed: the secret key mints the Testing Token and the sign-in
// token, the other two are the dedicated test user.
const REQUIRED = [
  "CLERK_SECRET_KEY",
  "E2E_CLERK_USER_EMAIL",
  "E2E_CLERK_USER_PASSWORD",
] as const

// Set by e2e.yml where the secrets must exist (pushes and same-repo pull
// requests, not forks or Dependabot): missing credentials then fail instead of
// skipping, so a misconfigured repository cannot go green with no sign-in run.
export const REQUIRE_SIGN_IN_VARIABLE = "E2E_REQUIRE_SIGN_IN"

function missingVariables(env: Env) {
  // GitHub passes an unset secret to a step as an empty string.
  return REQUIRED.filter((name) => !env[name]?.trim())
}

// Placeholders from compose.defaults.env: the API boots on them but no real
// Clerk call can succeed, so a signed-in run would fail deep in the browser
// with misleading errors. Names only, never values.
const PLACEHOLDERS = {
  CLERK_SECRET_KEY: /^sk_test_placeholder$/,
  CLERK_ISSUER: /^https:\/\/placeholder\.clerk\.accounts\.dev$/,
} satisfies Record<string, RegExp>

function placeholderVariables(env: Env) {
  return Object.entries(PLACEHOLDERS)
    .filter(([name]) => Boolean(env[name]?.trim()))
    .filter(([name, pattern]) => pattern.test(env[name]!.trim()))
    .map(([name]) => name)
}

export function readClerkE2eCredentials(env: Env): ClerkE2eCredentials | null {
  if (missingVariables(env).length > 0) return null
  return {
    email: env.E2E_CLERK_USER_EMAIL!.trim(),
    password: env.E2E_CLERK_USER_PASSWORD!,
  }
}

export type ClerkE2eDecision =
  | { action: "run" }
  | { action: "skip"; reason: string }
  | { action: "fail"; message: string }

// Messages name variables, never values.
export function decideClerkE2e(env: Env): ClerkE2eDecision {
  const missing = missingVariables(env)
  const placeholders = placeholderVariables(env)
  if (missing.length === 0 && placeholders.length === 0) {
    return { action: "run" }
  }
  const problems = [
    missing.length > 0 ? `missing or empty: ${missing.join(", ")}` : "",
    placeholders.length > 0
      ? `still the compose placeholder: ${placeholders.join(", ")}`
      : "",
  ].filter(Boolean)
  if (env[REQUIRE_SIGN_IN_VARIABLE] === "true") {
    return {
      action: "fail",
      message: `${REQUIRE_SIGN_IN_VARIABLE} is true but these variables are ${problems.join("; ")}.`,
    }
  }
  return {
    action: "skip",
    reason: `Clerk sign-in e2e is not configured (${problems.join("; ")}); fork pull requests and local runs without them skip these specs.`,
  }
}
