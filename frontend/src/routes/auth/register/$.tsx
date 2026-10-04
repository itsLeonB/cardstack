import { createFileRoute } from "@tanstack/react-router"
import { SignUpPage } from "@/components/auth/sign-up-page"
import { redirectSearchSchema } from "@/lib/route-guard"
import { pageHead } from "@/lib/site"

// The steps Clerk's path-routed component navigates to beneath `/auth/register`
// (`verify-email-address`, `sso-callback`, ...) render the same component, which picks
// the step from the address.
export const Route = createFileRoute("/auth/register/$")({
  head: () => pageHead("Create an account"),
  validateSearch: redirectSearchSchema,
  component: SignUpPage,
})
