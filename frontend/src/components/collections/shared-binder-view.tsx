import { useState } from "react"
import { keepPreviousData } from "@tanstack/react-query"
import { notFound } from "@tanstack/react-router"
import { Button } from "@/components/ui/button"
import { BinderPageGrid } from "@/components/collections/binder-page-grid"
import { useGetSharedCollectionPage } from "@/generated/endpoints/shared/shared"
import type { BinderHeader } from "@/generated/models"
import { pageOf } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

// The owner's binder, read-only: one Page at a time, with no Slot panel, Leaf,
// Layout or pending-review controls. A Page the binder does not have is thrown
// as not-found, so it replaces the whole shared page rather than sitting inside it.
export function SharedBinderView({
  profileSlug,
  collectionSlug,
  binder,
}: {
  profileSlug: string
  collectionSlug: string
  binder: BinderHeader
}) {
  const [requestedPage, setRequestedPage] = useState(1)
  // Keeps the previous Page on screen while the next one loads.
  const query = useGetSharedCollectionPage(
    profileSlug,
    collectionSlug,
    requestedPage,
    { query: { placeholderData: keepPreviousData } }
  )
  if (query.data?.status === 404) throw notFound()

  const page = query.data?.status === 200 ? query.data.data.data : undefined
  const totalPages = pageOf(binder.layout.leafCount, "back")
  const loadError = errorDetail(query.data, NETWORK_ERROR)

  // Steps from the requested Page, not the shown one, so two quick taps move two Pages.
  function stepPage(delta: number) {
    setRequestedPage((current) =>
      Math.min(Math.max(current + delta, 1), totalPages)
    )
  }

  if (!page) {
    return query.isError || loadError ? (
      <div role="alert" className="flex flex-col items-start gap-2">
        <p className="text-sm text-destructive">{loadError ?? NETWORK_ERROR}</p>
        <Button variant="outline" onClick={() => void query.refetch()}>
          Retry
        </Button>
      </div>
    ) : (
      <p className="text-sm">Loading this Page…</p>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1 text-sm">
        <p>Unidentified: {binder.unidentifiedCount}</p>
        <p>
          {binder.layout.rows} × {binder.layout.columns} slots per Page ·{" "}
          {binder.layout.leafCount} Leaves
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          aria-label="Previous page"
          disabled={requestedPage <= 1}
          onClick={() => stepPage(-1)}
        >
          Previous
        </Button>
        <p className="text-sm font-medium">
          Page {page.page} of {totalPages}
        </p>
        <Button
          variant="outline"
          aria-label="Next page"
          disabled={requestedPage >= totalPages}
          onClick={() => stepPage(1)}
        >
          Next
        </Button>
      </div>

      <section aria-label="Binder page" className="flex flex-col gap-3">
        <p className="text-sm">
          Leaf {page.leaf}, {page.side}
        </p>
        <BinderPageGrid page={page} />
      </section>
    </div>
  )
}
