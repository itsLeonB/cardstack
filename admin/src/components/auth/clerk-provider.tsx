import { ClerkProvider } from "@clerk/react"
import { ClerkBridge } from "./clerk-bridge"

/**
 * Wraps the app in Clerk. With no publishable key it renders a plain error
 * instead of the app: running on, unauthenticated, would make a misconfigured
 * deploy look like a Guest's not-found page.
 */
export function AppClerkProvider({ children }: { children: React.ReactNode }) {
  const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY

  if (!publishableKey) {
    return (
      <p role="alert" className="p-6 text-sm">
        Sign-in is not configured: set VITE_CLERK_PUBLISHABLE_KEY (see
        admin/.env.example) and rebuild.
      </p>
    )
  }

  return (
    <ClerkProvider publishableKey={publishableKey}>
      <ClerkBridge />
      {children}
    </ClerkProvider>
  )
}
