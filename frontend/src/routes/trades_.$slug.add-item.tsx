import { useCallback, useState } from "react"
import type { FormEvent } from "react"
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import {
  createFileRoute,
  useLocation,
  useNavigate,
  useRouter,
} from "@tanstack/react-router"

import { InfiniteCardResults } from "@/components/catalog/infinite-card-results"
import { CatalogFilterPanel } from "@/components/catalog/filter-panel"
import { useTradeSessionGate } from "@/components/trade/use-trade-session-gate"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { catalogFilterSchema } from "@/lib/catalog-search"
import type { CatalogFilters } from "@/lib/catalog-search"
import { NETWORK_ERROR } from "@/lib/collections"
import {
  useInfiniteCardResultsProps,
  useInfiniteCatalogCards,
} from "@/lib/infinite-catalog-cards"
import { loadTradeForSession } from "@/lib/trade-loader"
import { NOINDEX_META, pageHead } from "@/lib/site"
import { useIsGuest } from "@/lib/session"
import {
  useListCatalogFacets,
  useListCatalogSeries,
} from "@/generated/endpoints/catalog/catalog"
import { getListCollectionsQueryOptions } from "@/generated/endpoints/collections/collections"
import {
  getGetTradeQueryKey,
  useAddTradeItem,
} from "@/generated/endpoints/trades/trades"
import type { CardSummary } from "@/generated/models"

const TITLE = "Add a Card"
const LOCKED_MESSAGE = "trade items can only change while negotiating"
const MAX_QUANTITY = 2147483647

export const Route = createFileRoute("/trades_/$slug/add-item")({
  // Like the Trade detail page, the loader sends the Clerk token, which only
  // exists in the browser.
  ssr: false,
  validateSearch: catalogFilterSchema,
  loader: ({ context: { queryClient }, params }) =>
    loadTradeForSession(queryClient, params.slug),
  head: () => ({ meta: [...pageHead(TITLE).meta, NOINDEX_META] }),
  component: AddTradeItemPage,
})

function AddTradeItemPage() {
  const { slug } = Route.useParams()
  const { trade, sessionVersion } = Route.useLoaderData()
  const gate = useTradeSessionGate({ trade, sessionVersion, title: TITLE })
  if (gate || !trade || trade === "missing") return gate

  const canAddItem = trade.isParticipant && trade.status === "negotiating"

  return (
    <PageContainer variant="wide">
      <Breadcrumbs
        crumbs={[
          {
            label: "Trades",
            link: trade.isParticipant ? { to: "/trades" } : undefined,
          },
          {
            label: "Trade",
            link: { to: "/trades/$slug", params: { slug } },
          },
          { label: "Add a Card" },
        ]}
      />
      <PageHeader
        title={TITLE}
        description="Find a Card in the catalog, then choose the Collection it leaves and how many copies you offer."
      />
      {canAddItem ? (
        <TradeItemPicker key={`${sessionVersion}:${slug}`} slug={slug} />
      ) : (
        <p role="alert" className="text-sm text-destructive">
          {LOCKED_MESSAGE}
        </p>
      )}
    </PageContainer>
  )
}

function TradeItemPicker({ slug }: { slug: string }) {
  const search = Route.useSearch()
  const router = useRouter()
  const navigate = useNavigate({ from: Route.fullPath })
  const queryClient = useQueryClient()
  const isGuest = useIsGuest()
  const href = useLocation({ select: (location) => location.href })

  const [cardId, setCardId] = useState("")
  const [cardName, setCardName] = useState("")
  const [collectionId, setCollectionId] = useState("")
  const [quantity, setQuantity] = useState("1")
  const [error, setError] = useState("")

  const seriesQuery = useListCatalogSeries()
  const facetsQuery = useListCatalogFacets(search, {
    query: { placeholderData: keepPreviousData, enabled: !isGuest },
  })
  const cardsQuery = useInfiniteCatalogCards(search)
  const cardResults = useInfiniteCardResultsProps(cardsQuery)
  const collectionsQuery = useQuery(getListCollectionsQueryOptions())
  const addMutation = useAddTradeItem()

  const series =
    seriesQuery.data?.status === 200
      ? (seriesQuery.data.data.data.series ?? [])
      : []
  const facets =
    facetsQuery.data?.status === 200 ? facetsQuery.data.data.data : undefined
  const collections =
    collectionsQuery.data?.status === 200
      ? (collectionsQuery.data.data.data ?? [])
      : []
  const collectionsFailed =
    collectionsQuery.isError ||
    (collectionsQuery.data !== undefined &&
      collectionsQuery.data.status !== 200)

  async function updateSearch(patch: Partial<CatalogFilters>) {
    await navigate({ search: (prev) => ({ ...prev, ...patch }) })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  async function clearSearch() {
    await navigate({ search: {} })
    window.scrollTo({ top: 0, behavior: "instant" })
  }

  const renderControl = useCallback(
    (card: CardSummary) => (
      <Button
        type="button"
        size="sm"
        variant={cardId === card.id ? "default" : "outline"}
        aria-label={`Offer ${card.name}, ${card.expansionSet.code} No. ${card.localId}`}
        aria-pressed={cardId === card.id}
        onClick={() => {
          setCardId(card.id)
          setCardName(card.name)
          setError("")
        }}
      >
        {cardId === card.id ? "Selected" : "Offer"}
      </Button>
    ),
    [cardId]
  )

  function addItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const count = Number(quantity)
    if (!cardId) {
      setError("Choose a Card from the search results.")
      return
    }
    if (!collectionId) {
      setError("Choose a source Collection.")
      return
    }
    if (!Number.isInteger(count) || count < 1 || count > MAX_QUANTITY) {
      setError("Quantity must be at least 1.")
      return
    }
    setError("")
    addMutation.mutate(
      {
        slug,
        data: { cardId, quantity: count, sourceCollectionId: collectionId },
      },
      {
        onSuccess: (response) => {
          if (response.status === 200) {
            // The add response is the fresh Trade view; seed it so the detail
            // page's polling read starts from the item just added.
            queryClient.setQueryData(getGetTradeQueryKey(slug), response)
            void router
              .invalidate()
              .then(() => router.history.push(`/trades/${slug}`))
          } else {
            setError(
              response.data.detail ??
                "Could not add this Card. Check the Trade and try again."
            )
          }
        },
        onError: () => setError(NETWORK_ERROR),
      }
    )
  }

  return (
    <>
      <CatalogFilterPanel
        search={search}
        facets={facets}
        series={series}
        onChange={(patch) => void updateSearch(patch)}
        onClear={() => void clearSearch()}
        guest={isGuest}
        signInRedirect={href}
      />
      <InfiniteCardResults
        {...cardResults}
        emptyMessage="No cards match these filters."
        guest={isGuest}
        signInRedirect={href}
        renderControl={renderControl}
      />
      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-lg">Offer a Card</h2>
          </CardTitle>
          {cardName && (
            <p className="text-sm text-muted-foreground">
              Chosen Card: {cardName}
            </p>
          )}
        </CardHeader>
        <CardContent>
          <form
            onSubmit={addItem}
            noValidate
            className="grid gap-4 sm:grid-cols-2"
          >
            <div className="grid gap-1">
              <label htmlFor="picker-collection">Source Collection</label>
              <Select
                value={collectionId === "" ? null : collectionId}
                onValueChange={(value) => setCollectionId(value ?? "")}
                items={[
                  { label: "Choose a Collection", value: null },
                  ...collections.map((collection) => ({
                    label: collection.title,
                    value: collection.id,
                  })),
                ]}
              >
                <SelectTrigger id="picker-collection" className="w-full">
                  <SelectValue placeholder="Choose a Collection" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={null}>Choose a Collection</SelectItem>
                  {collections.map((collection) => (
                    <SelectItem key={collection.id} value={collection.id}>
                      {collection.title}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {collectionsQuery.data?.status === 200 &&
                collections.length === 0 && (
                  <p className="text-sm text-muted-foreground">
                    Create a Collection before adding a Card.
                  </p>
                )}
            </div>
            <div className="grid gap-1">
              <label htmlFor="picker-quantity">Quantity</label>
              <Input
                id="picker-quantity"
                type="number"
                min={1}
                max={MAX_QUANTITY}
                step={1}
                value={quantity}
                onChange={(event) => setQuantity(event.target.value)}
              />
            </div>
            {error && (
              <p
                role="alert"
                className="text-sm text-destructive sm:col-span-2"
              >
                {error}
              </p>
            )}
            <Button
              type="submit"
              disabled={
                addMutation.isPending ||
                collectionsQuery.isPending ||
                !collections.length
              }
              className="sm:col-span-2 sm:justify-self-start"
            >
              {addMutation.isPending ? "Adding..." : "Add Card"}
            </Button>
          </form>
        </CardContent>
      </Card>
      {collectionsFailed && (
        <div
          role="alert"
          className="flex flex-col items-start gap-2 text-sm text-destructive"
        >
          <p>Could not load your Collections.</p>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => void collectionsQuery.refetch()}
          >
            Retry Collections
          </Button>
        </div>
      )}
    </>
  )
}
