import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { ReviewPhoto } from "@/components/admin/review-photo"
import { Button } from "@/components/ui/button"
import {
  getListReviewQueueQueryKey,
  useListReviewQueue,
  useRejectReview,
} from "@/generated/endpoints/admin/admin"
import type {
  ReviewItem,
  ReviewItemKind,
  ReviewItemSource,
} from "@/generated/models"

// The API's default page size (ADR-0027 leaves paging to the list endpoint).
const PAGE_SIZE = 24

const KIND_LABEL = {
  switched: "Switched to another candidate",
  reprint: "Print from Other prints",
  placed: "Placed with no match offered",
} satisfies Record<ReviewItemKind, string>

const REJECT_FAILED = "Could not reject the photo. Try again."

const SOURCE_LABEL = {
  card_scan: "Card scan",
  binder_page: "Binder page",
} satisfies Record<ReviewItemSource, string>

/**
 * The Corrections whose photos wait for review, a page at a time, oldest first
 * (the order the API gives). Only Reject lives here: approval comes later.
 */
export function ReviewQueue() {
  const [page, setPage] = useState(1)
  const queue = useListReviewQueue({ page, limit: PAGE_SIZE })
  const envelope = queue.data?.status === 200 ? queue.data.data : undefined

  if (queue.isPending) return <p role="status">Loading the queue…</p>
  if (!envelope) {
    return (
      <div role="alert" className="flex flex-col items-start gap-3">
        <p>Could not load the review queue.</p>
        <Button variant="outline" onClick={() => queue.refetch()}>
          Retry
        </Button>
      </div>
    )
  }

  const items = envelope.data ?? []
  const { total, limit } = envelope.meta
  const pages = Math.max(1, Math.ceil(total / limit))

  return (
    <div className="flex flex-col gap-6">
      {items.length === 0 ? (
        <p>Nothing is waiting for review.</p>
      ) : (
        <ul className="flex flex-col gap-6">
          {items.map((item) => (
            <li key={item.id}>
              <ReviewItemCard item={item} />
            </li>
          ))}
        </ul>
      )}
      <nav aria-label="Queue pages" className="flex items-center gap-3">
        <Button
          variant="outline"
          disabled={page <= 1}
          onClick={() => setPage(page - 1)}
        >
          Previous
        </Button>
        <span>{`Page ${page} of ${pages}`}</span>
        <Button
          variant="outline"
          disabled={page >= pages}
          onClick={() => setPage(page + 1)}
        >
          Next
        </Button>
      </nav>
    </div>
  )
}

/**
 * One waiting photo beside the Card the user chose, with the scores the matcher
 * gave. Rejecting it asks the API; the queue is then fetched again, so the photo
 * leaves the list (or the Admin is told why it is not there to reject).
 */
function ReviewItemCard({ item }: { item: ReviewItem }) {
  const queryClient = useQueryClient()
  const reject = useRejectReview()
  const [message, setMessage] = useState<string | null>(null)
  const headingId = `review-${item.id}`

  function refreshQueue() {
    void queryClient.invalidateQueries({
      queryKey: getListReviewQueueQueryKey(),
    })
  }

  function handleReject() {
    setMessage(null)
    reject.mutate(
      { id: item.id },
      {
        onSuccess: ({ status }) => {
          if (status === 204) return refreshQueue()
          if (status === 404) {
            setMessage("This photo is no longer waiting for review.")
            return refreshQueue()
          }
          if (status === 409) {
            setMessage("Another Admin already decided this photo.")
            return refreshQueue()
          }
          setMessage(REJECT_FAILED)
        },
        onError: () => setMessage(REJECT_FAILED),
      }
    )
  }

  return (
    <section
      aria-labelledby={headingId}
      className="grid gap-6 rounded-xl border p-4 md:grid-cols-2"
    >
      <div className="flex flex-col gap-3">
        <h2 id={headingId} className="text-lg font-medium">
          {KIND_LABEL[item.kind]}
        </h2>
        <ReviewPhoto item={item} />
      </div>
      <div className="flex flex-col gap-4">
        <img
          src={item.card.imageUrl}
          alt={`${item.card.name} catalog image`}
          className="w-40 rounded-md"
        />
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt className="text-muted-foreground">Card</dt>
          <dd>{item.card.name}</dd>
          <dt className="text-muted-foreground">Set and number</dt>
          <dd>{`${item.card.expansionSet.code} ${item.card.localId}`}</dd>
          <dt className="text-muted-foreground">Source</dt>
          <dd>{SOURCE_LABEL[item.source]}</dd>
          <dt className="text-muted-foreground">Recorded</dt>
          <dd>{new Date(item.createdAt).toLocaleString()}</dd>
        </dl>
        <Scores scores={item.scores} />
        <div className="flex flex-col items-start gap-2">
          <Button
            variant="destructive"
            disabled={reject.isPending}
            onClick={handleReject}
          >
            Reject
          </Button>
          {message && (
            <p role="alert" className="text-sm">
              {message}
            </p>
          )}
        </div>
      </div>
    </section>
  )
}

function Scores({ scores }: { scores: ReviewItem["scores"] }) {
  if (!scores || scores.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No scores were recorded.</p>
    )
  }
  return (
    <div className="flex flex-col gap-1 text-sm">
      <h3 className="font-medium">Matcher scores</h3>
      <ul className="flex flex-col gap-1">
        {scores.map((score) => (
          <li key={score.cardId} className="flex justify-between gap-4">
            <span className="font-mono">{score.cardId}</span>
            <span className="tabular-nums">{score.score.toFixed(3)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
