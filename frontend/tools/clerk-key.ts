// Build step: a Vercel production build without Clerk's publishable key must
// fail, since the deployed app would show a "not configured" error instead of
// signing anyone in. Other builds (local, CI, previews) only warn.

interface BuildEnv {
  VITE_CLERK_PUBLISHABLE_KEY?: string
  VERCEL_ENV?: string
}

export function clerkKeyCheck(env: BuildEnv): "ok" | "missing" | "fatal" {
  if (env.VITE_CLERK_PUBLISHABLE_KEY) return "ok"
  return env.VERCEL_ENV === "production" ? "fatal" : "missing"
}

if (process.argv[1]?.endsWith("clerk-key.ts")) {
  const result = clerkKeyCheck(process.env)
  if (result === "fatal") {
    console.error(
      "[clerk-key] VITE_CLERK_PUBLISHABLE_KEY must be set for production builds"
    )
    process.exit(1)
  } else if (result === "missing") {
    console.warn(
      "[clerk-key] VITE_CLERK_PUBLISHABLE_KEY is unset: the built app will show a sign-in configuration error"
    )
  }
}
