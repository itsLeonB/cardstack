import type { ReactNode } from "react"
import { useSyncExternalStore } from "react"
import { useRouter } from "@tanstack/react-router"

import { SignInLink } from "@/components/auth/sign-in-link"
import { NotFoundView } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { clerkAuth } from "@/lib/clerk-auth"
import { useIsGuest } from "@/lib/session"
import type { TradeView } from "@/generated/models"

// The fallback a Trade route shows until its loader data belongs to the current
// session and a readable Trade is present. Returns null when the page can render.
export function useTradeSessionGate({
  trade,
  sessionVersion,
  title,
}: {
  trade: TradeView | "missing" | null
  sessionVersion: number
  title: string
}): ReactNode | null {
  const currentVersion = useSyncExternalStore(
    clerkAuth.subscribeSession,
    clerkAuth.currentSessionVersion,
    clerkAuth.currentSessionVersion
  )
  const router = useRouter()
  const isGuest = useIsGuest()

  if (sessionVersion !== currentVersion)
    return (
      <PageContainer>
        <PageHeader title={title} />
        <p role="status">Loading Trade...</p>
      </PageContainer>
    )
  if (trade === "missing") {
    const { pathname, searchStr } = router.state.location
    return (
      <NotFoundView resource="Trade">
        {isGuest && (
          <p className="text-sm">
            <SignInLink redirect={pathname + searchStr}>Sign in</SignInLink> to
            see this Trade.
          </p>
        )}
      </NotFoundView>
    )
  }
  if (!trade)
    return (
      <PageContainer>
        <PageHeader title={title} />
        <p role="alert" className="text-sm text-destructive">
          Could not load this Trade.
        </p>
        <div>
          <Button
            variant="outline"
            onClick={() => {
              if (clerkAuth.currentSessionVersion() === sessionVersion)
                void router.invalidate()
            }}
          >
            Retry
          </Button>
        </div>
      </PageContainer>
    )
  return null
}
