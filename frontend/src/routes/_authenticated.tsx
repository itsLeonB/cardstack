import { createFileRoute, Outlet } from "@tanstack/react-router"
import { NOINDEX_META } from "@/lib/site"
import { requireAuth } from "@/lib/route-guard"

/**
 * Pathless layout: groups every protected route under one `beforeLoad`
 * guard so new protected routes just need to live under `_authenticated/`
 * (see `src/routes/_authenticated/account.tsx`) to be covered by it.
 */
export const Route = createFileRoute("/_authenticated")({
  beforeLoad: requireAuth,
  // Clerk's state exists only in the browser (SPA); a server render of a
  // private route would have no way to run the guard.
  ssr: false,
  head: () => ({ meta: [NOINDEX_META] }),
  component: () => <Outlet />,
})
