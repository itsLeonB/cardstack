import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ScanScreen } from "./scan-screen"
import { getListMasterInventoryQueryKey } from "@/generated/endpoints/inventory/inventory"
import type { FrameSource } from "@/lib/frame-source"
import type { CardSummary } from "@/generated/models"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { loadDraft, saveDraft } from "@/lib/draft-addition"
import type { ResolvedScan } from "@/lib/draft-addition"
import type * as TanStackRouter from "@tanstack/react-router"

// The hint's link only needs its text here; the router is not under test.
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

// Only the network is faked (`fetch`). The Collection holds `held`; the bulk
// update applies every item unless `declines` lists it. The draft is seeded
// through local storage, so these tests start at the Review step.
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
// What the next bulk request does: answer normally, a server error, or no answer at all.
let bulkMode: "ok" | "error" | "lost" = "ok"
let bulks: { cardId: string; quantity: number }[][] = []
// A binder Collection's Slots: `cardTotal` of its `layout.capacity` are taken. Null for a plain Collection.
let binder: { cardTotal: number } | null = null
let placements: { cardId: string; quantity: number }[][] = []
// What the next placement does: answer normally, a server error (nothing placed), or commit and lose the answer.
let placementMode: "ok" | "error" | "lost" = "ok"
const layout = { rows: 1, columns: 5, leafCount: 1, capacity: 10 }

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
        data: {
          id: "col-1",
          title: "Binder",
          description: "",
          cardCount: 10,
          maxCardCount: 0,
          binder: binder ? layout : null,
        },
      })
    )
  }
  if (url.pathname === "/collections/col-1/binder/pages/1") {
    return Promise.resolve(
      json({
        data: {
          page: 1,
          leaf: 1,
          side: "front",
          layout,
          cardTotal: binder?.cardTotal ?? 0,
          unidentifiedCount: 0,
          slots: [],
        },
      })
    )
  }
  if (url.pathname === "/collections/col-1/binder/placements") {
    // SAFETY: the only POST this fake receives is the placement's.
    const { items } = JSON.parse(String(init?.body)) as {
      items: { cardId: string; quantity: number }[]
    }
    placements.push(items)
    if (placementMode === "error")
      return Promise.resolve(json({ title: "boom", detail: "Try later." }, 503))
    const results = items.map(({ cardId, quantity }) => {
      const placed = Math.min(quantity, layout.capacity - binder!.cardTotal)
      const declined = quantity - placed
      binder!.cardTotal += placed
      held.set(cardId, (held.get(cardId) ?? 0) + placed)
      return declined > 0
        ? {
            cardId,
            placed,
            declined,
            reason: "not_enough_empty_slots",
            message: "not enough empty Slots",
          }
        : { cardId, placed, declined }
    })
    // The write commits, but the answer never arrives.
    if (placementMode === "lost")
      return Promise.reject(new TypeError("offline"))
    return Promise.resolve(json({ data: results }))
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
  captureFrame: async () => new Blob(),
  frameFromFile: async () => new Blob(),
}

const seed = (scans: ResolvedScan[]) => saveDraft("user-1", "col-1", scans)
const scan = (n: number, quantity = 1): ResolvedScan => ({
  cardId: `card-${n}`,
  quantity,
  candidateIds: [],
})

const added = vi.fn()

function renderScreen(
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
) {
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanScreen
        userId="user-1"
        collectionId="col-1"
        source={frames}
        onAdded={added}
      />
    </QueryClientProvider>
  )
}

// Opens the Review step from the sheet, once the seeded Scans have restored.
async function openReview(queryClient?: QueryClient) {
  renderScreen(queryClient)
  await waitFor(() =>
    expect(
      screen
        .getByRole("button", { name: "Review and add" })
        .hasAttribute("disabled")
    ).toBe(false)
  )
  await userEvent.click(screen.getByRole("button", { name: "Review and add" }))
  return screen.findByRole("heading", { name: "Review and add" })
}

// GET requests the fake has answered for a path, so a test can see a refetch.
const readsOf = (path: string) =>
  vi
    .mocked(fetch)
    .mock.calls.filter(
      ([input, init]) =>
        (init?.method ?? "GET") === "GET" && String(input).includes(path)
    ).length

const reviewRow = async (name: string) => {
  const row = (await screen.findByText(name)).closest("li")
  if (!row) throw new Error(`no row for ${name}`)
  return row
}

beforeEach(() => {
  held = new Map()
  declines = new Map()
  bulkMode = "ok"
  bulks = []
  binder = null
  placements = []
  placementMode = "ok"
  added.mockClear()
  vi.stubGlobal("fetch", vi.fn(fakeApi))
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
  URL.createObjectURL = vi.fn(() => "blob:photo")
  URL.revokeObjectURL = vi.fn()
})
afterEach(() => {
  cleanup()
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe("the review rows", () => {
  it("shows one row per Card, with its Scans' quantities added together", async () => {
    await seed([scan(1), scan(1), scan(2, 3)])
    held.set("card-1", 4)
    await openReview()

    const row = await reviewRow("Card 1")
    expect(within(row).getByText("+2")).toBeTruthy()
    expect(await within(row).findByText("Holds 4")).toBeTruthy()
    expect(screen.getAllByRole("listitem")).toHaveLength(2)
  })

  it("does not let the quantities or the rows change here", async () => {
    await seed([scan(1)])
    await openReview()

    expect(screen.queryByRole("button", { name: /Raise quantity/ })).toBeNull()
    expect(screen.queryByRole("button", { name: /Remove/ })).toBeNull()
  })
})

describe("a decline reason that is out of date", () => {
  it("goes once the declined Card's Scan is edited", async () => {
    await seed([scan(1), scan(2)])
    declines.set("card-2", "capacity_exceeded")
    await openReview()
    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )
    expect(
      await screen.findByText("Not added: Declined: capacity_exceeded")
    ).toBeTruthy()

    await userEvent.click(
      screen.getByRole("button", { name: "Back to scanning" })
    )
    await userEvent.click(
      await screen.findByRole("button", { name: /^Draft \(/ })
    )
    await userEvent.click(
      await screen.findByRole("button", { name: "Raise quantity of Card 2" })
    )
    await userEvent.click(
      screen.getByRole("button", { name: "Review and add" })
    )

    const row = await reviewRow("Card 2")
    expect(within(row).getByText("+2")).toBeTruthy()
    expect(screen.queryByText(/Not added/)).toBeNull()
  })
})

describe("adding to the Collection", () => {
  it("sends absolute targets, held plus added, and returns to the Collection when all is applied", async () => {
    await seed([scan(1), scan(1), scan(2, 3)])
    held.set("card-1", 4)
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    await waitFor(() => expect(added).toHaveBeenCalledOnce())
    expect(bulks[0]).toEqual(
      expect.arrayContaining([
        { cardId: "card-1", quantity: 6 },
        { cardId: "card-2", quantity: 3 },
      ])
    )
    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })

  it("keeps a declined Card's Scans with the reason, and stays on the Review step", async () => {
    await seed([scan(1), scan(2)])
    declines.set("card-2", "capacity_exceeded")
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    const declined = await reviewRow("Card 2")
    expect(
      await within(declined).findByText(
        "Not added: Declined: capacity_exceeded"
      )
    ).toBeTruthy()
    expect(screen.queryByText("Card 1")).toBeNull()
    expect(added).not.toHaveBeenCalled()
    expect(await loadDraft("user-1", "col-1")).toEqual([scan(2)])
  })

  it("keeps the draft and says so when the request fails", async () => {
    await seed([scan(1)])
    bulkMode = "error"
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(screen.getByText(/Your draft is unchanged/)).toBeTruthy()
    expect(added).not.toHaveBeenCalled()
    expect(await loadDraft("user-1", "col-1")).toEqual([scan(1)])
  })

  it("does not add twice when the answer to a commit is lost", async () => {
    await seed([scan(1, 2)])
    bulkMode = "lost"
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )
    await screen.findByRole("alert")
    bulkMode = "ok"
    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    await waitFor(() => expect(added).toHaveBeenCalledOnce())
    expect(bulks[0]).toEqual([{ cardId: "card-1", quantity: 2 }])
    expect(bulks[1]).toEqual([{ cardId: "card-1", quantity: 2 }])
    expect(held.get("card-1")).toBe(2)
  })
})

describe("adding to a binder", () => {
  beforeEach(() => {
    binder = { cardTotal: 0 }
  })

  it("places each copy in an empty Slot, one line per Card, and returns to the Collection", async () => {
    await seed([scan(1), scan(1), scan(2, 3)])
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    await waitFor(() => expect(added).toHaveBeenCalledOnce())
    expect(placements).toEqual([
      [
        { cardId: "card-1", quantity: 2 },
        { cardId: "card-2", quantity: 3 },
      ],
    ])
    expect(bulks).toEqual([])
    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })

  it("keeps the copies that found no empty Slot, with the reason, and removes the placed ones", async () => {
    binder = { cardTotal: 8 }
    await seed([scan(1), scan(2, 2)])
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    const declined = await reviewRow("Card 2")
    expect(
      await within(declined).findByText("Not added: not enough empty Slots")
    ).toBeTruthy()
    expect(within(declined).getByText("+1")).toBeTruthy()
    expect(within(declined).getByText("binder view")).toBeTruthy()
    expect(screen.queryByText("Card 1")).toBeNull()
    expect(added).not.toHaveBeenCalled()
    expect(await loadDraft("user-1", "col-1")).toEqual([scan(2, 1)])
  })

  it("shows the binder's empty Slots instead of the Collection's limit", async () => {
    binder = { cardTotal: 8 }
    await seed([scan(1)])
    await openReview()

    expect(await screen.findByText("Empty Slots: 2 of 10")).toBeTruthy()
    expect(screen.queryByText(/Collection total/)).toBeNull()
  })

  it("refetches the binder Page and the entries, and invalidates Master Inventory, after a placement", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const inventory = getListMasterInventoryQueryKey()
    queryClient.setQueryData(inventory, { status: 200, data: { data: [] } })
    await seed([scan(1)])
    await openReview(queryClient)
    const before = {
      page: readsOf("/binder/pages/1"),
      entries: readsOf("/collections/col-1/entries"),
    }

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    await waitFor(() => expect(added).toHaveBeenCalledOnce())
    await waitFor(() =>
      expect(readsOf("/binder/pages/1")).toBeGreaterThan(before.page)
    )
    expect(readsOf("/collections/col-1/entries")).toBeGreaterThan(
      before.entries
    )
    expect(queryClient.getQueryState(inventory)?.isInvalidated).toBe(true)
  })

  it("says the add could not be confirmed when the answer is lost, keeps the draft, and shows the Slots the binder now has", async () => {
    await seed([scan(1, 2)])
    placementMode = "lost"
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(screen.getByText(/could not be confirmed/)).toBeTruthy()
    expect(
      screen.getByText(/Check the binder before adding again/)
    ).toBeTruthy()
    expect(await screen.findByText("Empty Slots: 8 of 10")).toBeTruthy()
    expect(added).not.toHaveBeenCalled()
    expect(await loadDraft("user-1", "col-1")).toEqual([scan(1, 2)])
  })

  it("says nothing was added when the server refuses the placement", async () => {
    await seed([scan(1, 2)])
    placementMode = "error"
    await openReview()

    await userEvent.click(
      screen.getByRole("button", { name: "Add to Collection" })
    )

    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(screen.getByText(/Your draft is unchanged/)).toBeTruthy()
    expect(screen.queryByText(/could not be confirmed/)).toBeNull()
    expect(await screen.findByText("Empty Slots: 10 of 10")).toBeTruthy()
    expect(added).not.toHaveBeenCalled()
    expect(await loadDraft("user-1", "col-1")).toEqual([scan(1, 2)])
  })
})
