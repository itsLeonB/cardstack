import { createFileRoute } from "@tanstack/react-router"
import { SignUpPage } from "@/components/auth/sign-up-page"
import { redirectSearchSchema } from "@/lib/route-guard"
import { pageHead } from "@/lib/site"

// The steps Clerk's path-routed component navigates to beneath `/auth/register`
// (`verify-email-address`, `sso-callback`, ...) render the same component, which picks
// the step from the address. Alone this file would also match the bare path, but
// typed `to="/auth/register"` links only type-check against the index route.
export const Route = createFileRoute("/auth/register/$")({
  head: () => pageHead("Create an account"),
  validateSearch: redirectSearchSchema,
  component: SignUpPage,
})
