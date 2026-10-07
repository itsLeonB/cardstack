import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ScanScreen } from "./scan-screen"
import type { FrameSource } from "@/lib/frame-source"
import type { CardSummary } from "@/generated/models"
import { loadDraft, saveDraft } from "@/lib/draft-addition"
import { getListCollectionEntriesInfiniteQueryKey } from "@/generated/endpoints/inventory/inventory"

// Only the network is faked (`fetch`): the generated client, the query cache
// and the draft store are real. The Collection holds `held`; the bulk update
// applies every item unless it is listed in `declines`.
const cardOf = (n: number): CardSummary => ({
  id: `card-${n}`,
  name: `Card ${n}`,
  localId: String(n),
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "SCE", name: "Starter Set", imageUrl: "" },
  rarity: { id: "r-1", code: "RR", name: "Double Rare" },
})

let held = new Map<string, number>()
let declines = new Map<string, "capacity_exceeded" | "card_not_found">()
let collection = { cardCount: 10, maxCardCount: 0 }
// What the next bulk request does: answer normally, a server error, or no answer at all.
let bulkMode: "ok" | "error" | "lost" = "ok"
let lookups: string[][] = []
let bulks: { cardId: string; quantity: number }[][] = []

interface JsonBody {
  data?: unknown
  meta?: unknown
  title?: string
  detail?: string
}
const json = (body: JsonBody, status = 200) =>
  new Response(JSON.stringify(body), { status })

function fakeApi(input: RequestInfo | URL, init?: RequestInit) {
  const url = new URL(String(input))
  if (url.pathname === "/catalog/cards") {
    const data = url.searchParams
      .getAll("cardId")
      .map((id) => cardOf(Number(id.replace("card-", ""))))
    return Promise.resolve(
      json({ data, meta: { total: data.length, page: 1, limit: 100 } })
    )
  }
  if (url.pathname === "/collections/col-1") {
    return Promise.resolve(
      json({
        data: { id: "col-1", title: "Binder", description: "", ...collection },
      })
    )
  }
  if (url.pathname === "/collections/col-1/entries") {
    if (init?.method === "PATCH") {
      // SAFETY: the only PATCH body this fake receives is the bulk update's.
      const { items } = JSON.parse(String(init.body)) as {
        items: { cardId: string; quantity: number }[]
      }
      bulks.push(items)
      if (bulkMode === "error")
        return Promise.resolve(
          json({ title: "boom", detail: "Try later." }, 503)
        )
      const results = items.map(({ cardId, quantity }) => {
        const reason = declines.get(cardId)
        if (reason)
          return {
            cardId,
            quantity: held.get(cardId) ?? 0,
            status: "declined",
            reason,
            message: `Declined: ${reason}`,
          }
        held.set(cardId, quantity)
        return { cardId, quantity, status: "applied" }
      })
      // The write commits, but the answer never arrives.
      if (bulkMode === "lost") return Promise.reject(new TypeError("offline"))
      return Promise.resolve(json({ data: results }))
    }
    const ids = url.searchParams.getAll("cardId")
    lookups.push(ids)
    const data = ids
      .filter((id) => (held.get(id) ?? 0) > 0)
      .map((id) => ({
        card: cardOf(Number(id.replace("card-", ""))),
        quantity: held.get(id),
      }))
    return Promise.resolve(
      json({ data, meta: { total: data.length, page: 1, limit: 100 } })
    )
  }
  return Promise.resolve(json({ title: "unexpected" }, 500))
}

const frames: FrameSource = {
  camera: "unavailable",
  videoRef: () => {},
  captureCamera: async () => new Blob(),
  fromFile: async () => new Blob(),
}

let queryClient: QueryClient
const added = vi.fn()

function renderScreen() {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanScreen collectionId="col-1" source={frames} onAdded={added} />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  held = new Map()
  declines = new Map()
  collection = { cardCount: 10, maxCardCount: 0 }
  bulkMode = "ok"
  lookups = []
  bulks = []
  added.mockClear()
  vi.stubGlobal("fetch", vi.fn(fakeApi))
})
afterEach(() => {
  cleanup()
  localStorage.clear()
  vi.unstubAllGlobals()
})

const rowOf = async (name: string) => {
  const row = (await screen.findByText(name)).closest("li")
  if (!row) throw new Error(`no row for ${name}`)
  return within(row)
}

async function openReview() {
  await userEvent.click(
    await screen.findByRole("button", { name: "Review and add" })
  )
  await screen.findByRole("heading", { name: "Review and add" })
}
const commit = () =>
  userEvent.click(screen.getByRole("button", { name: "Add to Collection" }))

describe("the review step", () => {
  beforeEach(() => {
    saveDraft("col-1", [
      { cardId: "card-1", quantity: 2 },
      { cardId: "card-2", quantity: 1 },
    ])
    held.set("card-1", 3)
  })

  it("lists each card with the quantity to add and what the Collection holds", async () => {
    renderScreen()
    await openReview()
    const one = await rowOf("Card 1")
    await vi.waitFor(() => expect(one.getByText("Holds 3")).toBeTruthy())
    expect(one.getByText("+2")).toBeTruthy()
    const two = await rowOf("Card 2")
    expect(two.getByText("Holds 0")).toBeTruthy()
    expect(two.getByText("+1")).toBeTruthy()
  })

  it("warns about capacity without blocking the request", async () => {
    collection = { cardCount: 10, maxCardCount: 11 }
    renderScreen()
    await openReview()
    expect(await screen.findByText(/13 of 11 cards/)).toBeTruthy()
    expect(screen.getByText(/over the limit/i)).toBeTruthy()
    await commit()
    await vi.waitFor(() => expect(bulks).toHaveLength(1))
  })

  it("shows the running total with no warning while under the limit", async () => {
    collection = { cardCount: 10, maxCardCount: 50 }
    renderScreen()
    await openReview()
    expect(await screen.findByText(/13 of 50 cards/)).toBeTruthy()
    expect(screen.queryByText(/over the limit/i)).toBeNull()
  })

  it("sends current plus added as absolute targets, with edits made in review", async () => {
    renderScreen()
    await openReview()
    await userEvent.click(
      (await rowOf("Card 2")).getByRole("button", {
        name: "Raise quantity of Card 2",
      })
    )
    await commit()
    await vi.waitFor(() => expect(added).toHaveBeenCalled())
    expect(bulks).toEqual([
      [
        { cardId: "card-1", quantity: 5 },
        { cardId: "card-2", quantity: 2 },
      ],
    ])
  })

  it("clears the draft and returns to the Collection when nothing is declined", async () => {
    renderScreen()
    await openReview()
    await commit()
    await vi.waitFor(() => expect(added).toHaveBeenCalledTimes(1))
    expect(loadDraft("col-1")).toEqual([])
  })

  it("refreshes the Collection's cached entries and counts", async () => {
    renderScreen()
    const key = getListCollectionEntriesInfiniteQueryKey("col-1")
    queryClient.setQueryData(key, { pages: [], pageParams: [] })
    await openReview()
    await commit()
    await vi.waitFor(() => expect(added).toHaveBeenCalled())
    expect(queryClient.getQueryState(key)?.isInvalidated).toBe(true)
  })

  it("drops removed rows from the request and lets the user go back", async () => {
    renderScreen()
    await openReview()
    await userEvent.click(
      (await rowOf("Card 2")).getByRole("button", { name: "Remove Card 2" })
    )
    await commit()
    await vi.waitFor(() => expect(bulks).toHaveLength(1))
    expect(bulks[0]).toEqual([{ cardId: "card-1", quantity: 5 }])
  })

  it("goes back to scanning with the draft intact", async () => {
    renderScreen()
    await openReview()
    await userEvent.click(
      screen.getByRole("button", { name: "Back to scanning" })
    )
    expect(
      await screen.findByRole("button", { name: "Review and add" })
    ).toBeTruthy()
    expect(loadDraft("col-1")).toHaveLength(2)
  })
})

describe("a mixed result", () => {
  beforeEach(() => {
    saveDraft("col-1", [
      { cardId: "card-1", quantity: 2 },
      { cardId: "card-2", quantity: 1 },
      { cardId: "card-3", quantity: 4 },
    ])
    declines.set("card-2", "capacity_exceeded")
    declines.set("card-3", "card_not_found")
  })

  it("removes applied rows, keeps declined ones with their reason, and stays put", async () => {
    renderScreen()
    await openReview()
    await commit()
    expect(await screen.findByText(/capacity_exceeded/)).toBeTruthy()
    expect(screen.getByText(/card_not_found/)).toBeTruthy()
    expect(screen.queryByText("Card 1")).toBeNull()
    expect(loadDraft("col-1")).toEqual([
      { cardId: "card-2", quantity: 1 },
      { cardId: "card-3", quantity: 4 },
    ])
    expect(added).not.toHaveBeenCalled()
  })

  it("lets the user drop a declined row and finish the rest", async () => {
    renderScreen()
    await openReview()
    await commit()
    await screen.findByText(/capacity_exceeded/)
    await userEvent.click(
      (await rowOf("Card 2")).getByRole("button", { name: "Remove Card 2" })
    )
    await userEvent.click(
      (await rowOf("Card 3")).getByRole("button", { name: "Remove Card 3" })
    )
    expect(loadDraft("col-1")).toEqual([])
  })

  it("retries a declined row as current plus added once it fits", async () => {
    renderScreen()
    await openReview()
    await commit()
    await screen.findByText(/capacity_exceeded/)
    declines.delete("card-2")
    await commit()
    await vi.waitFor(() => expect(bulks).toHaveLength(2))
    expect(bulks[1]).toEqual([
      { cardId: "card-2", quantity: 1 },
      { cardId: "card-3", quantity: 4 },
    ])
  })
})

describe("a failed request", () => {
  beforeEach(() => {
    saveDraft("col-1", [{ cardId: "card-1", quantity: 2 }])
    held.set("card-1", 3)
  })

  it("keeps the draft and shows a message", async () => {
    bulkMode = "error"
    renderScreen()
    await openReview()
    await commit()
    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(screen.getByRole("alert").textContent).toMatch(/Try later/)
    expect(loadDraft("col-1")).toEqual([{ cardId: "card-1", quantity: 2 }])
    expect(added).not.toHaveBeenCalled()
  })

  it("does not double-add when a retry follows a write whose answer was lost", async () => {
    bulkMode = "lost"
    renderScreen()
    await openReview()
    await commit()
    expect(await screen.findByRole("alert")).toBeTruthy()
    // The first write committed, so the Collection now holds 5.
    bulkMode = "ok"
    await commit()
    await vi.waitFor(() => expect(added).toHaveBeenCalled())
    expect(bulks).toEqual([
      [{ cardId: "card-1", quantity: 5 }],
      [{ cardId: "card-1", quantity: 5 }],
    ])
  })
})

describe("a large draft", () => {
  it("batches the lookup and the update at 100 cards", async () => {
    saveDraft(
      "col-1",
      Array.from({ length: 101 }, (_, i) => ({
        cardId: `card-${i + 1}`,
        quantity: 1,
      }))
    )
    renderScreen()
    await openReview()
    await commit()
    await vi.waitFor(() => expect(added).toHaveBeenCalled())
    expect(lookups.every((ids) => ids.length <= 100)).toBe(true)
    expect(bulks.map((items) => items.length).sort()).toEqual([1, 100])
  })
})
