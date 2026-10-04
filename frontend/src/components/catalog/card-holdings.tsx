import { Link } from "@tanstack/react-router"
import { SignInLink } from "@/components/auth/sign-in-link"
import { useListCardHoldings } from "@/generated/endpoints/inventory/inventory"
import { Skeleton } from "@/components/ui/skeleton"
import { useSession } from "@/lib/session"

/** "Your collections": which of the signed-in user's Collections hold this Card, and how many. */
export function CardHoldings({
  cardId,
  loginRedirect,
}: {
  cardId: string
  /** Path (with search) the sign-in prompt returns to. */
  loginRedirect?: string
}) {
  const { isAuthenticated, isLoading } = useSession()
  const query = useListCardHoldings(cardId, {
    query: { enabled: isAuthenticated },
  })

  let body
  if (isLoading || (isAuthenticated && query.isPending)) {
    body = (
      <Skeleton className="h-10 w-full" aria-label="Loading your collections" />
    )
  } else if (!isAuthenticated) {
    body = (
      <p className="text-sm text-muted-foreground">
        <SignInLink redirect={loginRedirect}>Sign in</SignInLink> to see which
        of your Collections hold this Card.
      </p>
    )
  } else if (query.isError || query.data?.status !== 200) {
    const detail =
      query.data && query.data.status !== 200
        ? query.data.data.detail
        : undefined
    body = (
      <p role="alert" className="text-sm text-destructive">
        {detail ?? "Could not load your collections. Please try again."}
      </p>
    )
  } else if (!query.data.data.data?.length) {
    body = (
      <p className="text-sm text-muted-foreground">
        Not in any of your Collections
      </p>
    )
  } else {
    body = (
      <ul className="flex flex-col divide-y rounded-lg border">
        {query.data.data.data?.map(({ collection, quantity }) => (
          <li
            key={collection.id}
            className="flex items-center justify-between gap-3 px-3 py-2 text-sm"
          >
            <Link
              to="/collections/$collectionId"
              params={{ collectionId: collection.id }}
              className="font-medium underline-offset-2 hover:underline"
            >
              {collection.name}
            </Link>
            <span className="text-muted-foreground">×{quantity}</span>
          </li>
        ))}
      </ul>
    )
  }

  return (
    <section aria-label="Your collections" className="flex flex-col gap-3">
      <h2 className="font-heading text-lg font-medium">Your collections</h2>
      {body}
    </section>
  )
}
