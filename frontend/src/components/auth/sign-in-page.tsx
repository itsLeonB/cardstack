import { SignIn } from "@clerk/react"
import { useSearch } from "@tanstack/react-router"
import { withRedirect } from "@/lib/route-guard"
import { AuthPage } from "./auth-page"

/**
 * Clerk's sign-in, mounted with path routing: its sub-steps (`factor-one`,
 * `sso-callback`, ...) live under `/auth/login/*`, which the splat route
 * `login/$.tsx` renders. `forceRedirectUrl` (not the fallback) so a
 * `redirect_url` query param on the address can never pick the destination.
 * `redirect` was validated by the route (`redirectSearchSchema`).
 */
export function SignInPage() {
  const { redirect } = useSearch({ strict: false })
  return (
    <AuthPage title="Log in">
      <SignIn
        routing="path"
        path="/auth/login"
        signUpUrl={withRedirect("/auth/register", redirect)}
        forceRedirectUrl={redirect ?? "/account"}
      />
    </AuthPage>
  )
}
