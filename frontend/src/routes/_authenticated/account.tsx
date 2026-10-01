import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { useLogoutMutation, useSession } from "@/lib/session"

/**
 * Minimal protected page demonstrating the `_authenticated` guard — later
 * tickets (Collections/Inventory) will add real protected content here and
 * elsewhere under `_authenticated/`.
 */
export const Route = createFileRoute("/_authenticated/account")({
  component: AccountPage,
})

function AccountPage() {
  const { user } = useSession()
  const navigate = useNavigate()
  const logoutMutation = useLogoutMutation()

  function handleLogout() {
    logoutMutation.mutate(undefined, {
      onSuccess: (response) => {
        if (response.status === 204) {
          void navigate({ to: "/login" })
        }
      },
    })
  }

  return (
    <div className="flex flex-col gap-4 p-6">
      <h1 className="font-medium">Account</h1>
      {user && <p>Logged in as {user.email}</p>}
      <Link to="/collections" className="w-fit text-sm text-primary underline">
        My Collections
      </Link>
      <Link to="/inventory" className="w-fit text-sm text-primary underline">
        Master Inventory
      </Link>
      <Button
        variant="outline"
        className="w-fit"
        disabled={logoutMutation.isPending}
        onClick={handleLogout}
      >
        {logoutMutation.isPending ? "Logging out..." : "Log out"}
      </Button>
    </div>
  )
}
