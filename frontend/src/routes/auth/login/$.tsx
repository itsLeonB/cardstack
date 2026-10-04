import { createFileRoute } from "@tanstack/react-router"
import { SignInPage } from "@/components/auth/sign-in-page"
import { redirectSearchSchema } from "@/lib/route-guard"
import { pageHead } from "@/lib/site"

// The steps Clerk's path-routed component navigates to beneath `/auth/login`
// (`factor-one`, `sso-callback`, ...) render the same component, which picks
// the step from the address. Alone this file would also match the bare path, but
// typed `to="/auth/login"` links only type-check against the index route.
export const Route = createFileRoute("/auth/login/$")({
  head: () => pageHead("Log in"),
  validateSearch: redirectSearchSchema,
  component: SignInPage,
})
