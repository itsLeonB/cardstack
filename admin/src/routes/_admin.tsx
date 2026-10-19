import { Outlet, createFileRoute } from "@tanstack/react-router"
import { requireAdmin } from "@/lib/admin-gate"

/**
 * Pathless layout over every screen of the Admin app: one `beforeLoad` gate,
 * so a screen added under `_admin/` is covered without a check of its own.
 */
export const Route = createFileRoute("/_admin")({
  beforeLoad: requireAdmin,
  // Clerk's state exists only in the browser (SPA), as for the user app's
  // `_authenticated` layout.
  ssr: false,
  component: () => <Outlet />,
})
