import { afterEach, describe, expect, it, vi } from "vitest"
import { act, cleanup, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionCardResults } from "./collection-card-results"
import { stubGridLayout } from "@/test-grid-layout"
import type { CardSummary } from "@/generated/models"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params: _params, to: _to, ...props }: any) => (
      <a {...props}>{children}</a>
    ),
  }
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const cardOf = (n: number): CardSummary => ({
  id: `card-${n}`,
  name: `Card ${n}`,
  localId: String(n),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "SCE", name: "Starter", imageUrl: "" },
  rarity: { id: "r-1", code: "RR", name: "Double Rare" },
})

/** Answers a Collection entries lookup like the API: only the asked-for ids, at most `limit` rows. */
function stubEntriesApi() {
  const lookups: { ids: string[]; limit: number }[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input))
      const ids = url.searchParams.getAll("cardId")
      const limit = Number(url.searchParams.get("limit") ?? 24)
      lookups.push({ ids, limit })
      const rows = ids.slice(0, limit).map((id) => ({
        card: cardOf(Number(id.replace("card-", ""))),
        quantity: 2,
      }))
      return new Response(
        JSON.stringify({
          data: rows,
          meta: { total: rows.length, page: 1, limit },
        }),
        { status: 200, headers: { "content-type": "application/json" } }
      )
    })
  )
  return lookups
}

function results(count: number) {
  const cards = Array.from({ length: count }, (_, index) => cardOf(index + 1))
  return (
    <CollectionCardResults
      collectionId="col-1"
      cards={cards}
      total={count}
      isPending={false}
      isError={false}
      emptyMessage="none"
      hasNextPage={false}
      isFetching={false}
      onLoadMore={vi.fn()}
    />
  )
}

async function settle() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20))
  })
}

describe("CollectionCardResults across loaded pages", () => {
  it("keeps a quantity control on every tile after page 2 appends", async () => {
    const grid = stubGridLayout()
    stubEntriesApi()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const tree = (count: number) => (
      <QueryClientProvider client={client}>
        {results(count)}
      </QueryClientProvider>
    )

    const { rerender } = render(tree(60))
    await settle()
    expect(screen.getByLabelText("Quantity of Card 1")).toBeTruthy()

    // Page 2 appends; the reported symptom is that its tiles have no control.
    rerender(tree(120))
    await settle()
    await act(async () => grid.scrollTo(100_000))
    await settle()

    const last = screen.getAllByLabelText(/^Quantity of Card \d+$/)
    const names = last.map((el) => el.getAttribute("aria-label"))
    expect(names).toContain("Quantity of Card 120")
  })
})
