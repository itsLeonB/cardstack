import { createFileRoute, Outlet } from "@tanstack/react-router"
import { requireAuth } from "@/lib/route-guard"

/**
 * Pathless layout: groups every protected route under one `beforeLoad`
 * guard so new protected routes just need to live under `_authenticated/`
 * (see `src/routes/_authenticated/account.tsx`) to be covered by it.
 */
export const Route = createFileRoute("/_authenticated")({
  beforeLoad: requireAuth,
  component: () => <Outlet />,
})
