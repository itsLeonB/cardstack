import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useLogoutMutation, useSession } from "@/lib/session"

/** Settings-style page, reached from the user menu. Navigation lives in the shell, not here. */
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
    <PageContainer>
      <PageHeader title="Account" description="Your sign-in details and session." />

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Profile</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="flex flex-col gap-1 text-sm">
            <dt className="font-medium">Email</dt>
            <dd className="break-all">{user?.email}</dd>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Session</h2>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-start gap-3">
          <p className="text-sm">Sign out of Cardstack on this device.</p>
          <Button
            variant="outline"
            disabled={logoutMutation.isPending}
            onClick={handleLogout}
          >
            {logoutMutation.isPending ? "Logging out..." : "Log out"}
          </Button>
        </CardContent>
      </Card>
    </PageContainer>
  )
}
