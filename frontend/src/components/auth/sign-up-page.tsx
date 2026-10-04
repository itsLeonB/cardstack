import { SignUp } from "@clerk/react"
import { useSearch } from "@tanstack/react-router"
import { AFTER_SIGN_IN_PATH, LOGIN_PATH, REGISTER_PATH } from "@/lib/auth-paths"
import { withRedirect } from "@/lib/route-guard"
import { AuthPage } from "./auth-page"

/** Clerk's sign-up; see `SignInPage` for the routing and redirect choices. */
export function SignUpPage() {
  const { redirect } = useSearch({ strict: false })
  return (
    <AuthPage title="Create an account">
      <SignUp
        routing="path"
        path={REGISTER_PATH}
        signInUrl={withRedirect(LOGIN_PATH, redirect)}
        forceRedirectUrl={redirect ?? AFTER_SIGN_IN_PATH}
      />
    </AuthPage>
  )
}
