import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
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

      <section
        className="flex flex-col gap-3 rounded-4xl bg-card p-6 ring-1 ring-foreground/5"
        aria-labelledby="profile-heading"
      >
        <h2 id="profile-heading" className="font-heading text-lg font-medium">
          Profile
        </h2>
        <dl className="flex flex-col gap-1 text-sm">
          <dt className="font-medium">Email</dt>
          <dd className="break-all">{user?.email}</dd>
        </dl>
      </section>

      <section
        className="flex flex-col items-start gap-3 rounded-4xl bg-card p-6 ring-1 ring-foreground/5"
        aria-labelledby="session-heading"
      >
        <h2 id="session-heading" className="font-heading text-lg font-medium">
          Session
        </h2>
        <p className="text-sm">Sign out of Cardstack on this device.</p>
        <Button
          variant="outline"
          disabled={logoutMutation.isPending}
          onClick={handleLogout}
        >
          {logoutMutation.isPending ? "Logging out..." : "Log out"}
        </Button>
      </section>
    </PageContainer>
  )
}
