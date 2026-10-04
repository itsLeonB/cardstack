import { createFileRoute } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useSession, useSignOut } from "@/lib/session"

/** Settings-style page, reached from the user menu. Navigation lives in the shell, not here. */
export const Route = createFileRoute("/_authenticated/account")({
  head: () => pageHead("Account"),
  component: AccountPage,
})

function AccountPage() {
  const { user } = useSession()
  const { signOut, isPending } = useSignOut()

  return (
    <PageContainer>
      <PageHeader
        title="Account"
        description="Your sign-in details and session."
      />

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Profile</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="flex flex-col gap-1 text-sm">
            {user?.name && (
              <>
                <dt className="font-medium">Name</dt>
                <dd className="break-words">{user.name}</dd>
              </>
            )}
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
            disabled={isPending}
            onClick={() => void signOut()}
          >
            {isPending ? "Logging out..." : "Log out"}
          </Button>
        </CardContent>
      </Card>
    </PageContainer>
  )
}
