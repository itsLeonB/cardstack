import { useEffect } from "react"
import { useAuth, useClerk } from "@clerk/react"
import { clerkAuth } from "@/lib/clerk-auth"

/**
 * Renders nothing. Hands the loaded Clerk instance to `lib/clerk-auth.ts`, which
 * serves the code that runs outside React (the request wrapper, the route
 * guards, session-change handling).
 */
export function ClerkBridge() {
  const { isLoaded } = useAuth()
  const clerk = useClerk()

  useEffect(() => {
    if (!isLoaded) return
    clerkAuth.publish(clerk)
    return () => clerkAuth.publish(null)
  }, [isLoaded, clerk])

  return null
}
