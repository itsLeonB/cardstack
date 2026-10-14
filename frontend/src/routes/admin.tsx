import { createFileRoute, notFound } from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"
import { AdminPage } from "@/components/admin/admin-page"
import type { AuthGate } from "@/lib/clerk-auth"
import { NOINDEX_META, pageHead } from "@/lib/site"

/**
 * The catalog admin (ADR-0019). Linked from nowhere and left out of the
 * sitemap and search results; it exists only for whoever types the address.
 * The page is split into its own chunk by the router plugin, and the generated
 * client is imported on demand, so neither is in the main bundle.
 */
export const Route = createFileRoute("/admin")({
  head: () => ({ meta: [...pageHead("Catalog admin").meta, NOINDEX_META] }),
  beforeLoad: requireAdmin,
  // Clerk's state exists only in the browser (SPA), as for `_authenticated`.
  ssr: false,
  component: AdminPage,
})

/**
 * Asks the API whether the caller is an Admin: it answers 204 to one and 404 to
 * everyone else, and the page shows the ordinary not-found page to a 404. A
 * Guest is not even asked. Any other answer (429, 5xx) is a real failure, not
 * proof the page does not exist, so it surfaces as an error.
 */
async function requireAdmin({
  context,
}: {
  context: { auth: AuthGate; queryClient: QueryClient }
}) {
  if (!(await context.auth.isSignedIn())) throw notFound()

  const { getAdminCheckQueryOptions } =
    await import("@/generated/endpoints/admin/admin")
  const check = await context.queryClient.fetchQuery(
    getAdminCheckQueryOptions()
  )
  if (check.status === 404) throw notFound()
  if (check.status !== 204) {
    throw new Error(`Could not check admin access (status ${check.status})`)
  }
}
