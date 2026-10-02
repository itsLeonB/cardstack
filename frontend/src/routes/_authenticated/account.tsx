import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
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
    <PageContainer className="gap-4">
      <PageHeader title="Account" />
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
    </PageContainer>
  )
}
