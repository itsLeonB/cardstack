import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ScanScreen } from "./scan-screen"
import type { FrameSource } from "@/lib/frame-source"
import type { CardSummary, MatchCandidate } from "@/generated/models"
import { loadDraft, saveDraft } from "@/lib/draft-addition"

// Only the network is faked (`fetch`): the generated client, the query cache
// and the draft store are real. The frame source is the camera boundary.
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
const candidate = (n: number, score = 0.9): MatchCandidate => ({
  card: cardOf(n),
  score,
})

type MatchReply =
  { confident: boolean; candidates: MatchCandidate[] } | "fail" | "network"
let replies: MatchReply[] = []
let uploads: Blob[] = []
let catalogRequests: string[] = []
// Network conditions a test can set: requests that never answer, or a failing lookup.
let holdCatalog = false
let holdMatch = false
let catalogFails = false
// Card ids the catalog no longer has.
const gone = new Set<string>()

interface JsonBody {
  data?: unknown
  meta?: unknown
  title?: string
}
const json = (body: JsonBody, status = 200) =>
  new Response(JSON.stringify(body), { status })

function fakeApi(input: RequestInfo | URL, init?: RequestInit) {
  const url = new URL(String(input))
  if (url.pathname === "/scan/match") {
    if (holdMatch) return new Promise<Response>(() => {})
    if (init?.body instanceof Blob) uploads.push(init.body)
    const reply = replies.shift() ?? "fail"
    if (reply === "network") return Promise.reject(new TypeError("offline"))
    if (reply === "fail") return Promise.resolve(json({ title: "boom" }, 503))
    return Promise.resolve(json({ data: reply }))
  }
  if (url.pathname === "/catalog/cards") {
    catalogRequests.push(url.search)
    if (holdCatalog) return new Promise<Response>(() => {})
    if (catalogFails) return Promise.resolve(json({ title: "boom" }, 500))
    const ids = url.searchParams.getAll("cardId").filter((id) => !gone.has(id))
    const data = ids.map((id) => cardOf(Number(id.replace("card-", ""))))
    return Promise.resolve(
      json({ data, meta: { total: data.length, page: 1, limit: 100 } })
    )
  }
  return Promise.resolve(json({ title: "unexpected" }, 500))
}

const jpeg = () => new Blob(["jpeg"], { type: "image/jpeg" })
function source(overrides: Partial<FrameSource> = {}): FrameSource {
  return {
    camera: "ready",
    videoRef: () => {},
    captureCamera: async () => jpeg(),
    fromFile: async () => jpeg(),
    ...overrides,
  }
}

function renderScreen(collectionId = "col-1", frames = source()) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanScreen collectionId={collectionId} source={frames} />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  replies = []
  uploads = []
  catalogRequests = []
  holdCatalog = false
  holdMatch = false
  catalogFails = false
  gone.clear()
  vi.stubGlobal("fetch", vi.fn(fakeApi))
})
afterEach(() => {
  cleanup()
  localStorage.clear()
  vi.unstubAllGlobals()
})

// The row (`li`) that shows a card by name.
const rowOf = async (name: string | RegExp) => {
  const row = (await screen.findByText(name)).closest("li")
  if (!row) throw new Error(`no row for ${name}`)
  return row
}

const capture = () =>
  userEvent.click(screen.getByRole("button", { name: "Capture" }))

describe("a confident match", () => {
  it("uploads the JPEG, adds the card to the tray and shows it", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const status = await screen.findByRole("status")
    expect(within(status).getByText("Added Card 1")).toBeTruthy()
    expect(within(status).getByText(/Starter Set/)).toBeTruthy()
    expect(within(status).getByText(/No\. 1/)).toBeTruthy()
    expect(uploads[0]).toBeInstanceOf(Blob)
    expect(uploads[0].type).toBe("image/jpeg")

    const row = await screen.findByRole("listitem")
    expect(within(row).getByText("Card 1")).toBeTruthy()
    expect(within(row).getByText("1")).toBeTruthy()
  })

  it("undoes the added copy", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await userEvent.click(await screen.findByRole("button", { name: "Undo" }))
    expect(screen.queryByRole("listitem")).toBeNull()
    expect(loadDraft("col-1")).toEqual([])
  })

  it("merges a second scan of the same card into quantity 2", async () => {
    replies = [
      { confident: true, candidates: [candidate(1)] },
      { confident: true, candidates: [candidate(1)] },
    ]
    renderScreen()
    await capture()
    await screen.findByText("Added Card 1")
    await capture()
    await screen.findByText("Added Card 1")
    const rows = await screen.findAllByRole("listitem")
    expect(rows).toHaveLength(1)
    expect(within(rows[0]).getByText("2")).toBeTruthy()
  })
})

describe("a match that is not confident", () => {
  const unsure = () => ({
    confident: false,
    candidates: [candidate(1, 0.6), candidate(2, 0.58), candidate(3, 0.5)],
  })

  it("offers the candidates and adds the one the user taps", async () => {
    replies = [unsure()]
    renderScreen()
    await capture()
    await userEvent.click(
      await screen.findByRole("button", { name: /Card 2.*SCE.*2/ })
    )
    const row = await screen.findByRole("listitem")
    expect(within(row).getByText("Card 2")).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Skip" })).toBeNull()
  })

  it("leaves the draft untouched when the user skips", async () => {
    replies = [unsure()]
    renderScreen()
    await capture()
    await userEvent.click(await screen.findByRole("button", { name: "Skip" }))
    expect(screen.queryByRole("listitem")).toBeNull()
    expect(screen.queryByRole("button", { name: /Card 1/ })).toBeNull()
  })
})

describe("a failed match", () => {
  it.each(["fail", "network"] as const)(
    "shows a message and keeps the draft (%s)",
    async (reply) => {
      saveDraft("col-1", [{ cardId: "card-1", quantity: 2 }])
      replies = [reply]
      renderScreen()
      await capture()
      const alert = await screen.findByRole("alert")
      expect(alert.textContent).toMatch(/draft is unchanged/i)
      const row = await screen.findByRole("listitem")
      expect(within(row).getByText("2")).toBeTruthy()
    }
  )

  it("shows a message when the frame cannot be captured", async () => {
    renderScreen(
      "col-1",
      source({
        captureCamera: () => Promise.reject(new Error("not ready")),
      })
    )
    await capture()
    expect((await screen.findByRole("alert")).textContent).toMatch(/capture/i)
  })
})

describe("the tray", () => {
  beforeEach(() => {
    saveDraft("col-1", [
      { cardId: "card-1", quantity: 2 },
      { cardId: "card-2", quantity: 1 },
    ])
  })

  it("raises, lowers and removes a row", async () => {
    renderScreen()
    const row = async () => within(await rowOf("Card 1"))
    await userEvent.click(
      (await row()).getByRole("button", { name: "Raise quantity of Card 1" })
    )
    expect((await row()).getByText("3")).toBeTruthy()
    await userEvent.click(
      (await row()).getByRole("button", { name: "Lower quantity of Card 1" })
    )
    await userEvent.click(
      (await row()).getByRole("button", { name: "Lower quantity of Card 1" })
    )
    expect((await row()).getByText("1")).toBeTruthy()
    expect(
      (await row())
        .getByRole("button", { name: "Lower quantity of Card 1" })
        .hasAttribute("disabled")
    ).toBe(true)
    await userEvent.click(
      (await row()).getByRole("button", { name: "Remove Card 1" })
    )
    expect(screen.queryByText("Card 1")).toBeNull()
    expect(loadDraft("col-1")).toEqual([{ cardId: "card-2", quantity: 1 }])
  })

  it("discards the whole draft", async () => {
    renderScreen()
    await userEvent.click(
      await screen.findByRole("button", { name: "Discard draft" })
    )
    expect(screen.queryByRole("listitem")).toBeNull()
    expect(loadDraft("col-1")).toEqual([])
  })

  it("renders a restored draft through the cardId filter", async () => {
    renderScreen()
    expect(await screen.findByText("Card 1")).toBeTruthy()
    expect(await screen.findByText("Card 2")).toBeTruthy()
    const params = new URLSearchParams(catalogRequests[0])
    expect(params.getAll("cardId").sort()).toEqual(["card-1", "card-2"])
  })

  it("keeps a separate draft per Collection", async () => {
    renderScreen("col-2")
    expect(screen.queryByRole("listitem")).toBeNull()
    expect(screen.getByText(/Draft \(0 cards\)/)).toBeTruthy()
  })

  it("has a Review and add button", async () => {
    renderScreen()
    expect(
      await screen.findByRole("button", { name: "Review and add" })
    ).toBeTruthy()
  })
})

describe("the tray while cards load", () => {
  it("keeps loaded rows readable when a scan adds a row", async () => {
    saveDraft("col-1", [{ cardId: "card-1", quantity: 1 }])
    replies = [{ confident: true, candidates: [candidate(2)] }]
    renderScreen()
    await screen.findByText("Card 1")

    holdCatalog = true
    await capture()
    await screen.findByText("Added Card 2")

    expect(screen.queryByText("Loading…")).toBeNull()
    expect(screen.getByText("Card 1")).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Raise quantity of Card 1" })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Raise quantity of Card 2" })
    ).toBeTruthy()
  })

  it("marks a card the catalog no longer has, with controls still usable", async () => {
    gone.add("card-9")
    saveDraft("col-1", [
      { cardId: "card-1", quantity: 1 },
      { cardId: "card-9", quantity: 2 },
    ])
    renderScreen()
    const row = within(await rowOf(/unavailable/i))
    expect(screen.queryByText("Loading…")).toBeNull()
    await userEvent.click(
      row.getByRole("button", { name: "Raise quantity of card" })
    )
    expect(row.getByText("3")).toBeTruthy()
    await userEvent.click(row.getByRole("button", { name: "Remove card" }))
    expect(loadDraft("col-1")).toEqual([{ cardId: "card-1", quantity: 1 }])
  })

  it("marks every row unavailable when the lookup fails", async () => {
    catalogFails = true
    saveDraft("col-1", [{ cardId: "card-1", quantity: 1 }])
    renderScreen()
    const row = within(await rowOf(/unavailable/i))
    expect(screen.queryByText("Loading…")).toBeNull()
    expect(row.getByText("1")).toBeTruthy()
  })
})

describe("announcements", () => {
  it("keeps the Undo button out of the live region", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    const status = await screen.findByRole("status")
    await within(status).findByText("Added Card 1")
    expect(within(status).queryByRole("button")).toBeNull()
    expect(screen.getByRole("button", { name: "Undo" })).toBeTruthy()
  })

  it("announces the matching state", async () => {
    holdMatch = true
    renderScreen()
    await capture()
    expect(
      within(screen.getByRole("status")).getByText("Matching…")
    ).toBeTruthy()
  })

  it("announces how many candidates to choose from, buttons outside", async () => {
    replies = [
      {
        confident: false,
        candidates: [candidate(1, 0.6), candidate(2, 0.58)],
      },
    ]
    renderScreen()
    await capture()
    const status = await screen.findByRole("status")
    await within(status).findByText(/2 possible matches/)
    expect(within(status).queryByRole("button")).toBeNull()
  })
})

describe("the frame source", () => {
  it("explains a denied camera and still takes a photo file", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen("col-1", source({ camera: "denied" }))
    expect((await screen.findByRole("alert")).textContent).toMatch(
      /camera access was denied/i
    )
    expect(screen.queryByRole("button", { name: "Capture" })).toBeNull()

    await userEvent.upload(
      screen.getByLabelText("Choose a photo"),
      new File(["x"], "card.jpg", { type: "image/jpeg" })
    )
    expect(await screen.findByText("Added Card 1")).toBeTruthy()
  })

  it("says so when there is no camera", () => {
    renderScreen("col-1", source({ camera: "unavailable" }))
    expect(screen.getByRole("alert").textContent).toMatch(/no camera/i)
  })
})
