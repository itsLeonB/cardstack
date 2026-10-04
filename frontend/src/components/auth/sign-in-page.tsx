import { SignIn } from "@clerk/react"
import { useSearch } from "@tanstack/react-router"
import { AFTER_SIGN_IN_PATH, LOGIN_PATH, REGISTER_PATH } from "@/lib/auth-paths"
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
    <AuthPage>
      <SignIn
        routing="path"
        path={LOGIN_PATH}
        signUpUrl={withRedirect(REGISTER_PATH, redirect)}
        forceRedirectUrl={redirect ?? AFTER_SIGN_IN_PATH}
      />
    </AuthPage>
  )
}
