import { useSession } from "@/lib/session"

export type NavLink = {
  to: "/catalog" | "/collections" | "/inventory" | "/auth/login" | "/auth/register"
  label: string
  primary?: boolean
}

/**
 * Header and footer links for the current auth state. While the session is
 * still resolving only the always-public Catalog is shown, so neither state's
 * links flash in the wrong one.
 */
export function useNavLinks(): NavLink[] {
  const { isAuthenticated, isLoading } = useSession()
  const catalog: NavLink = { to: "/catalog", label: "Catalog" }

  if (isLoading) return [catalog]
  if (isAuthenticated) {
    return [
      catalog,
      { to: "/collections", label: "Collections" },
      { to: "/inventory", label: "Master Inventory" },
    ]
  }
  return [
    catalog,
    { to: "/auth/login", label: "Log in" },
    { to: "/auth/register", label: "Register", primary: true },
  ]
}
