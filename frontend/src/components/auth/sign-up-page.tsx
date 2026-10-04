import { SignUp } from "@clerk/react"
import { useSearch } from "@tanstack/react-router"
import { withRedirect } from "@/lib/route-guard"
import { AuthPage } from "./auth-page"

/** Clerk's sign-up; see `SignInPage` for the routing and redirect choices. */
export function SignUpPage() {
  const { redirect } = useSearch({ strict: false })
  return (
    <AuthPage title="Create an account">
      <SignUp
        routing="path"
        path="/auth/register"
        signInUrl={withRedirect("/auth/login", redirect)}
        forceRedirectUrl={redirect ?? "/account"}
      />
    </AuthPage>
  )
}
