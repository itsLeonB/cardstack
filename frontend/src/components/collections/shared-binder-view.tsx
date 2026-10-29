import { useState } from "react"
import { keepPreviousData } from "@tanstack/react-query"
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
  // Keeps the previous Page on screen while the next one loads. No Page stays
  // cached past this view, so a revisit reads each Page again.
  const query = useGetSharedCollectionPage(
    profileSlug,
    collectionSlug,
    requestedPage,
    { query: { placeholderData: keepPreviousData, gcTime: 0 } }
  )
  const missing = query.data?.status === 404
  const page = query.data?.status === 200 ? query.data.data.data : undefined
  const totalPages = pageOf(binder.layout.leafCount, "back")
  const loadError = missing ? undefined : errorDetail(query.data, NETWORK_ERROR)
  const { rows, columns, leafCount } = binder.layout

  // Steps from the requested Page, not the shown one, so two quick taps move two Pages.
  function stepPage(delta: number) {
    setRequestedPage((current) =>
      Math.min(Math.max(current + delta, 1), totalPages)
    )
  }

  let body: React.ReactNode
  if (missing) {
    // Inline rather than the root not-found: that would drop the route's noindex and nest a second h1.
    body = (
      <div className="flex flex-col gap-2">
        <h2 className="text-lg font-semibold">Page not found</h2>
        <p className="text-sm">This binder has no Page {requestedPage}.</p>
      </div>
    )
  } else if (query.isError || loadError) {
    body = (
      <div role="alert" className="flex flex-col items-start gap-2">
        <p className="text-sm text-destructive">{loadError ?? NETWORK_ERROR}</p>
        <Button variant="outline" onClick={() => void query.refetch()}>
          Retry
        </Button>
      </div>
    )
  } else if (!page) {
    body = <p className="text-sm">Loading this Page…</p>
  } else {
    body = (
      <section aria-label="Binder page" className="flex flex-col gap-3">
        <p className="text-sm">
          Leaf {page.leaf}, {page.side}
        </p>
        <BinderPageGrid page={page} />
      </section>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1 text-sm">
        <p>Unidentified: {binder.unidentifiedCount}</p>
        {page && (
          <p>
            Cards and Unidentified: {page.cardTotal} of {page.layout.capacity}{" "}
            slots
          </p>
        )}
        <p>
          {rows} × {columns} slots per Page · {leafCount}{" "}
          {leafCount === 1 ? "Leaf" : "Leaves"}
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
          Page {page?.page ?? requestedPage} of {totalPages}
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

      {body}
    </div>
  )
}
