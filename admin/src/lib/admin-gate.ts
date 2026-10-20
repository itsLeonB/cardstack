import { notFound } from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"
import { getAdminCheckQueryOptions } from "@/generated/endpoints/admin/admin"
import type { AuthGate } from "@/lib/clerk-auth"

/**
 * Asks the API whether the caller is an Admin: it answers 204 to one and 404 to
 * everyone else, and the ordinary not-found page answers the 404. A Guest is
 * not even asked. Any other answer (429, 5xx) is a real failure, not proof that
 * the app is not for this caller, so it surfaces as an error.
 */
export async function requireAdmin({
  context,
}: {
  context: { auth: AuthGate; queryClient: QueryClient }
}) {
  if (!(await context.auth.isSignedIn())) throw notFound()

  const check = await context.queryClient.fetchQuery(
    getAdminCheckQueryOptions()
  )
  if (check.status === 404) throw notFound()
  if (check.status !== 204) {
    throw new Error(`Could not check admin access (status ${check.status})`)
  }
}
