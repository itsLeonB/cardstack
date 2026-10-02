import { createFileRoute, Outlet } from "@tanstack/react-router"
import { AuthShell } from "@/components/auth/auth-shell"
import { requireGuest } from "@/lib/route-guard"

/**
 * Guest-only layout: everything under `auth/` (login, register, anything
 * added later) is guarded by one `beforeLoad` and rendered in `AuthShell`
 * instead of the main app shell (see `__root.tsx`).
 */
export const Route = createFileRoute("/auth")({
  beforeLoad: requireGuest,
  component: () => (
    <AuthShell>
      <Outlet />
    </AuthShell>
  ),
})
