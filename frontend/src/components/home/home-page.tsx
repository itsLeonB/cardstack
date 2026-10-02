import { PageContainer } from "@/components/layout/page-container"
import { Skeleton } from "@/components/ui/skeleton"
import { useSession } from "@/lib/session"
import { Dashboard } from "./dashboard"
import { GuestLanding } from "./guest-landing"

/**
 * `/` for everyone. While the session resolves it shows a skeleton, never the
 * landing, so a signed-in user doesn't see marketing copy flash. A failed or
 * unauthenticated session check is a guest.
 */
export function HomePage() {
  const { isAuthenticated, isLoading } = useSession()

  if (isLoading) return <HomeSkeleton />
  return isAuthenticated ? <Dashboard /> : <GuestLanding />
}

function HomeSkeleton() {
  return (
    <PageContainer role="status" aria-busy="true" aria-label="Loading">
      <Skeleton className="h-8 w-2/3" />
      <Skeleton className="h-28 w-full" />
      <Skeleton className="h-20 w-full" />
    </PageContainer>
  )
}
