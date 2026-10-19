// Build step: a production build without the API address would bake in the
// local fallback and call localhost from the deployed Admin app. Other builds
// (local, preview) keep the fallback and only warn.

interface BuildEnv {
  VITE_API_BASE_URL?: string
  VERCEL_ENV?: string
}

export function apiAddressCheck(env: BuildEnv): "ok" | "missing" | "fatal" {
  if (env.VITE_API_BASE_URL) return "ok"
  return env.VERCEL_ENV === "production" ? "fatal" : "missing"
}

if (process.argv[1]?.endsWith("api-address.ts")) {
  const result = apiAddressCheck(process.env)
  if (result === "fatal") {
    console.error(
      "[api-address] VITE_API_BASE_URL must be set for production builds"
    )
    process.exit(1)
  } else if (result === "missing") {
    console.warn(
      "[api-address] VITE_API_BASE_URL is unset: the built app will call http://localhost:8080"
    )
  }
}
