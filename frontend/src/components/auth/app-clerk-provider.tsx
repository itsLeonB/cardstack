import { ClerkProvider } from "@clerk/react"
import { useRouter } from "@tanstack/react-router"
import { clerkAppearance } from "@/lib/clerk-appearance"
import { AFTER_SIGN_IN_PATH, LOGIN_PATH, REGISTER_PATH } from "@/lib/auth-paths"
import { ClerkBridge } from "./clerk-bridge"

/**
 * Wraps the app in Clerk. With no publishable key it renders a plain error
 * instead of the app: running on, unauthenticated, would hide a misconfigured
 * deploy behind a working-looking guest site.
 */
export function AppClerkProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY

  if (!publishableKey) {
    return (
      <p role="alert" className="p-6 text-sm">
        Sign-in is not configured: set VITE_CLERK_PUBLISHABLE_KEY (see
        frontend/.env.example) and rebuild.
      </p>
    )
  }

  return (
    <ClerkProvider
      publishableKey={publishableKey}
      appearance={clerkAppearance}
      signInUrl={LOGIN_PATH}
      signUpUrl={REGISTER_PATH}
      signInFallbackRedirectUrl={AFTER_SIGN_IN_PATH}
      signUpFallbackRedirectUrl={AFTER_SIGN_IN_PATH}
      afterSignOutUrl="/"
      // The router, not a full page load, so Clerk's steps and redirects keep
      // the SPA's state. `history.push` takes the query string `navigate({ to })` can't.
      routerPush={(to) => router.history.push(to)}
      routerReplace={(to) => router.history.replace(to)}
    >
      <ClerkBridge />
      {children}
    </ClerkProvider>
  )
}
