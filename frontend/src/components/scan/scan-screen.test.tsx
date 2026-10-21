import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ScanScreen } from "./scan-screen"
import type { FrameSource } from "@/lib/frame-source"
import type { CardSummary, MatchCandidate } from "@/generated/models"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
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
const candidate = (n: number): MatchCandidate => ({
  card: cardOf(n),
  score: 0.9,
})

type Reply =
  { confident: boolean; candidates: MatchCandidate[]; empty?: boolean } | "fail"

// Replies come in order. In hold mode a match waits until the test answers it,
// and an aborted one is dropped from the queue of waiting answers.
let replies: Reply[] = []
let holding = false
let waiting: { answer: (reply: Reply) => void }[] = []
let matchCalls = 0
let aborted = 0
let uploads: Blob[] = []
let catalogRequests: string[] = []
let failLookup = new Set<string>()
let holdLookup = new Set<string>()
let reprintPool: CardSummary[] = []
let reprintCalls: string[] = []
let reprintFailures = 0

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
  if (url.pathname === "/scan/match") {
    matchCalls++
    if (init?.body) {
      // SAFETY: the match route sends the photo itself as the request body, a Blob.
      uploads.push(init.body as Blob)
    }
    if (holding)
      return new Promise<Response>((resolve, reject) => {
        const entry = {
          answer: (reply: Reply) =>
            resolve(
              reply === "fail"
                ? json({ title: "boom" }, 503)
                : json({ data: reply })
            ),
        }
        waiting.push(entry)
        init?.signal?.addEventListener("abort", () => {
          aborted++
          waiting = waiting.filter((w) => w !== entry)
          reject(new DOMException("aborted", "AbortError"))
        })
      })
    const reply = replies.shift() ?? "fail"
    return Promise.resolve(
      reply === "fail" ? json({ title: "boom" }, 503) : json({ data: reply })
    )
  }
  if (url.pathname === "/catalog/cards") {
    const ids = url.searchParams.getAll("cardId")
    catalogRequests.push(url.search)
    if (ids.some((id) => failLookup.has(id)))
      return Promise.resolve(json({ title: "boom" }, 500))
    // A lookup for one card that never answers, so a row cannot get its card from it.
    if (ids.length === 1 && holdLookup.has(ids[0]))
      return new Promise<Response>(() => {})
    const data = ids.map((id) => cardOf(Number(id.replace("card-", ""))))
    return Promise.resolve(
      json({ data, meta: { total: data.length, page: 1, limit: 100 } })
    )
  }
  if (url.pathname === "/scan/reprints") {
    reprintCalls.push(url.search)
    if (reprintFailures > 0) {
      reprintFailures--
      return Promise.resolve(json({ title: "boom" }, 500))
    }
    // Like the API: the sent ids are left out, then five a page from the offset.
    const sent = new Set(url.searchParams.getAll("cardIds"))
    const offset = Number(url.searchParams.get("offset") ?? 0)
    const pool = reprintPool.filter((card) => !sent.has(card.id))
    return Promise.resolve(
      json({
        data: {
          cards: pool.slice(offset, offset + 5),
          hasMore: offset + 5 < pool.length,
        },
      })
    )
  }
  return Promise.resolve(json({ title: "unexpected" }, 500))
}

// Answers the oldest held match.
const answer = (reply: Reply) => waiting.shift()!.answer(reply)

// jsdom's Blob does not survive the in-memory IndexedDB's structured clone, so
// photos here are Node's Blob, which the saved Draft Addition can hold.
// SAFETY: Node's Blob reads the same as the DOM Blob a photo is typed as.
const jpeg = () => new NodeBlob(["jpeg"], { type: "image/jpeg" }) as Blob
function source(overrides: Partial<FrameSource> = {}): FrameSource {
  return {
    camera: "ready",
    videoRef: () => {},
    captureCamera: async () => jpeg(),
    fromFile: async () => jpeg(),
    captureFrame: async () => jpeg(),
    frameFromFile: async () => jpeg(),
    ...overrides,
  }
}

function renderScreen(
  collectionId = "col-1",
  frames = source(),
  onAdded = () => {},
  userId = "user-1"
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanScreen
        userId={userId}
        collectionId={collectionId}
        source={frames}
        onAdded={onAdded}
      />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  replies = []
  holding = false
  waiting = []
  matchCalls = 0
  aborted = 0
  uploads = []
  catalogRequests = []
  failLookup = new Set()
  holdLookup = new Set()
  reprintPool = []
  reprintCalls = []
  reprintFailures = 0
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

const capture = () =>
  userEvent.click(screen.getByRole("button", { name: "Capture" }))

// The sheet opens collapsed (newest Scan only); most assertions need every row.
const expandSheet = async () =>
  userEvent.click(await screen.findByRole("button", { name: /^Draft \(/ }))

// The row (`li`) that shows a card by name.
const rowOf = async (name: string | RegExp) => {
  const row = (await screen.findByText(name)).closest("li")
  if (!row) throw new Error(`no row for ${name}`)
  return row
}

describe("capturing", () => {
  it("adds a Scan at once, before its match returns, and uploads the photo", async () => {
    holding = true
    renderScreen()
    await capture()

    const row = await rowOf("Matching…")
    expect(within(row).getByText("Matching…")).toBeTruthy()
    expect(uploads[0].size).toBeGreaterThan(0)
    expect(uploads[0].type).toBe("image/jpeg")
    // Capture stays usable while the match is out.
    expect(screen.getByRole("button", { name: "Capture" })).toHaveProperty(
      "disabled",
      false
    )
  })

  it("keeps three matches in flight and starts a fourth once one settles", async () => {
    holding = true
    renderScreen()
    await expandSheet()
    for (let i = 0; i < 4; i++) await capture()

    expect(await screen.findAllByText("Matching…")).toHaveLength(3)
    expect(screen.getByText("Queued")).toBeTruthy()
    expect(matchCalls).toBe(3)

    answer({ confident: true, candidates: [candidate(1)] })

    await waitFor(() => expect(screen.queryByText("Queued")).toBeNull())
    expect(matchCalls).toBe(4)
  })

  it("shows a confident match as its card, its menu offering the card, Other prints and the marks", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Card 1")
    expect(within(row).getByText(/Starter Set · No\. 1/)).toBeTruthy()
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    expect(
      await screen.findByRole("menuitemradio", { name: /Card 1/ })
    ).toBeTruthy()
    expect(screen.getByRole("menuitem", { name: "Other prints" })).toBeTruthy()
    expect(
      screen.getByRole("menuitemcheckbox", { name: "No card in the photo" })
    ).toBeTruthy()
  })

  it("shows an unsure match on its top candidate and lets the user pick another, keeping the quantity", async () => {
    replies = [
      {
        confident: false,
        candidates: [candidate(1), candidate(2), candidate(3)],
      },
    ]
    renderScreen()
    await capture()
    const row = await rowOf("Card 1")
    await userEvent.click(
      within(row).getByRole("button", { name: "Raise quantity of Card 1" })
    )

    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemradio", { name: /Card 3/ })
    )
    // The collapsed sheet shows one row; the open menu's items sit outside it.
    const updated = screen.getByRole("listitem")
    await waitFor(() =>
      expect(within(updated).getByText("Card 3")).toBeTruthy()
    )
    expect(within(updated).getByText("2")).toBeTruthy()
  })

  it("marks a failed match with its reason and retries from the photo held in memory", async () => {
    replies = ["fail", { confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Could not match this photo.")
    await userEvent.click(within(row).getByRole("button", { name: "Retry" }))
    expect(await rowOf("Card 1")).toBeTruthy()
    expect(matchCalls).toBe(2)
    expect(uploads[1]).toBe(uploads[0])
  })

  it("treats a match with no candidates as failed", async () => {
    replies = [{ confident: false, candidates: [] }]
    renderScreen()
    await capture()

    const row = await rowOf("No card matched.")
    expect(within(row).getByRole("button", { name: "Retry" })).toBeTruthy()
  })
})

describe("removing Scans", () => {
  it("drops a queued Scan without sending its request", async () => {
    holding = true
    renderScreen()
    await expandSheet()
    for (let i = 0; i < 4; i++) await capture()

    const queued = await rowOf("Queued")
    await userEvent.click(
      within(queued).getByRole("button", { name: /^Remove/ })
    )

    expect(screen.queryByText("Queued")).toBeNull()
    expect(matchCalls).toBe(3)
  })

  it("cancels the request of a loading Scan", async () => {
    holding = true
    renderScreen()
    await capture()

    const loading = await rowOf("Matching…")
    await userEvent.click(
      within(loading).getByRole("button", { name: /^Remove/ })
    )

    expect(aborted).toBe(1)
    expect(screen.queryByText("Matching…")).toBeNull()
  })
})

describe("the Review and add gate", () => {
  it("is disabled while a Scan is matching, and says how many", async () => {
    holding = true
    renderScreen()
    await capture()

    expect(screen.getByText("1 still matching")).toBeTruthy()
    expect(
      screen
        .getByRole("button", { name: "Review and add" })
        .hasAttribute("disabled")
    ).toBe(true)

    answer({ confident: true, candidates: [candidate(1)] })

    await rowOf("Card 1")
    expect(
      screen
        .getByRole("button", { name: "Review and add" })
        .hasAttribute("disabled")
    ).toBe(false)
  })

  it("is not blocked by a failed Scan", async () => {
    replies = ["fail", { confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Could not match this photo.")
    await capture()
    await rowOf("Card 1")
    await expandSheet()

    expect(screen.getByText("Could not match this photo.")).toBeTruthy()
    expect(
      screen
        .getByRole("button", { name: "Review and add" })
        .hasAttribute("disabled")
    ).toBe(false)
  })
})

describe("the sheet", () => {
  it("starts collapsed with the newest Scan, then lists every Scan when expanded", async () => {
    replies = [
      { confident: true, candidates: [candidate(1)] },
      { confident: true, candidates: [candidate(2)] },
    ]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await capture()
    await rowOf("Card 2")

    expect(screen.getAllByRole("listitem")).toHaveLength(1)
    await expandSheet()
    expect(screen.getAllByRole("listitem")).toHaveLength(2)
  })

  it("says so when there are no Scans", () => {
    renderScreen()

    expect(screen.getByText("Scanned cards will appear here.")).toBeTruthy()
  })

  it("discards the whole draft from the expanded sheet", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await expandSheet()

    await userEvent.click(screen.getByRole("button", { name: "Discard draft" }))

    expect(screen.getByText("Scanned cards will appear here.")).toBeTruthy()
    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })
})

describe("persistence", () => {
  it("restores resolved Scans with their alternatives after a reload", async () => {
    await saveDraft("user-1", "col-1", [
      { cardId: "card-2", quantity: 3, candidateIds: ["card-2", "card-4"] },
    ])
    renderScreen()
    await expandSheet()

    const row = await rowOf("Card 2")
    expect(within(row).getByText("3")).toBeTruthy()
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 2" })
    )
    expect(
      await screen.findByRole("menuitemradio", { name: /Card 4/ })
    ).toBeTruthy()
  })

  it("keeps only resolved Scans in storage, so pending and failed ones are lost on reload", async () => {
    holding = true
    replies = []
    renderScreen()
    await capture()
    await rowOf("Matching…")

    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })

  it("keeps a Scan's mark across a reload", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    const first = renderScreen()
    await capture()
    const row = await rowOf("Card 1")
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemcheckbox", {
        name: "No card in the photo",
      })
    )
    await waitFor(async () =>
      expect((await loadDraft("user-1", "col-1"))[0]?.mark).toBe("empty")
    )

    first.unmount()
    renderScreen()
    await expandSheet()
    expect(
      within(await rowOf("Card 1")).getByText("Marked: no card in the photo")
    ).toBeTruthy()
  })

  it("drops a Reprint choice when the Scan is marked", async () => {
    await saveDraft("user-1", "col-1", [
      {
        cardId: "card-2",
        quantity: 1,
        candidateIds: ["card-2", "card-4"],
        reprintPosition: 3,
      },
    ])
    renderScreen()
    await expandSheet()
    await userEvent.click(
      await screen.findByRole("button", { name: "Change card for Card 2" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemcheckbox", {
        name: "A card, but none of these",
      })
    )
    await waitFor(async () => {
      const [saved] = await loadDraft("user-1", "col-1")
      expect(saved?.mark).toBe("unidentified")
      expect(saved?.reprintPosition).toBeUndefined()
    })
  })

  it("keeps each Collection's draft separate", async () => {
    await saveDraft("user-1", "col-2", [
      { cardId: "card-9", quantity: 1, candidateIds: [] },
    ])
    renderScreen("col-1")

    expect(
      await screen.findByText("Scanned cards will appear here.")
    ).toBeTruthy()
  })

  it("keeps another user's draft out of this screen", async () => {
    await saveDraft("user-2", "col-1", [
      { cardId: "card-9", quantity: 1, candidateIds: [] },
    ])
    renderScreen("col-1", source(), () => {}, "user-1")

    expect(
      await screen.findByText("Scanned cards will appear here.")
    ).toBeTruthy()
    expect(await loadDraft("user-2", "col-1")).toHaveLength(1)
  })

  it("keeps a resolved Scan's photo and Detection as the match returned them", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Card 1")

    await waitFor(async () =>
      expect(await loadDraft("user-1", "col-1")).toHaveLength(1)
    )
    const [saved] = await loadDraft("user-1", "col-1")
    expect(saved.cardId).toBe("card-1")
    expect(await saved.photo?.text()).toBe("jpeg")
    expect(saved.detection).toEqual({
      data: { confident: true, candidates: [candidate(1)] },
    })
  })

  it("runs from memory, with nothing saved, when IndexedDB is missing", async () => {
    vi.stubGlobal("indexedDB", undefined)
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    expect(await rowOf("Card 1")).toBeTruthy()
  })
})

describe("announcements", () => {
  it("reads a repeat of the same match again", async () => {
    replies = [
      { confident: true, candidates: [candidate(1)] },
      { confident: true, candidates: [candidate(1)] },
    ]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    const status = screen.getByRole("status")
    const first = status.querySelector("span")

    await capture()
    await waitFor(() => expect(status.querySelector("span")).not.toBe(first))
    expect(status.textContent).toBe("Matched Card 1.")
  })

  it("does not announce the result of a Scan removed while its answer was on the way", async () => {
    holding = true
    renderScreen()
    await capture()
    const row = await rowOf("Matching…")

    answer({ confident: true, candidates: [candidate(1)] })
    fireEvent.click(within(row).getByRole("button", { name: /^Remove/ }))
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(screen.getByRole("status").textContent).not.toContain("Matched")
  })
})

describe("removing a Scan from the saved draft", () => {
  it("forgets the removed Scan and keeps the rest", async () => {
    replies = [
      { confident: true, candidates: [candidate(1)] },
      { confident: true, candidates: [candidate(2)] },
    ]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await capture()
    const row = await rowOf("Card 2")
    await waitFor(async () =>
      expect(await loadDraft("user-1", "col-1")).toHaveLength(2)
    )

    fireEvent.click(within(row).getByRole("button", { name: /^Remove/ }))

    await waitFor(async () =>
      expect((await loadDraft("user-1", "col-1")).map((s) => s.cardId)).toEqual(
        ["card-1"]
      )
    )
  })
})

describe("the candidates", () => {
  it("offers only the one candidate, with the marks, when the matcher offers a single candidate", async () => {
    replies = [{ confident: false, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Card 1")
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    expect(await screen.findAllByRole("menuitemradio")).toHaveLength(1)
    expect(
      screen.getByRole("menuitemcheckbox", { name: "No card in the photo" })
    ).toBeTruthy()
  })

  it("lists the one Card of a confident match as the only choice", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    await rowOf("Card 1")
    await userEvent.click(
      screen.getByRole("button", { name: "Change card for Card 1" })
    )
    expect(await screen.findAllByRole("menuitemradio")).toHaveLength(1)
  })

  it("shows the picked card at once, before the sheet's own lookup of it answers", async () => {
    await saveDraft("user-1", "col-1", [
      { cardId: "card-2", quantity: 1, candidateIds: ["card-2", "card-4"] },
    ])
    holdLookup = new Set(["card-4"])
    renderScreen()
    await expandSheet()
    await userEvent.click(
      await screen.findByRole("button", { name: "Change card for Card 2" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemradio", { name: /Card 4/ })
    )

    const row = screen.getByRole("listitem")
    expect(within(row).getByText("Card 4")).toBeTruthy()
    expect(within(row).queryByText("Loading…")).toBeNull()
  })
})

describe("other prints", () => {
  const openOtherPrints = async (name: string) => {
    await userEvent.click(
      screen.getByRole("button", { name: `Change card for ${name}` })
    )
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Other prints" })
    )
  }
  // A row's name runs into its set code ("Card 11SCE"), so a digit after the name means another Card.
  const printed = (name: string) => new RegExp(`^${name}(?!\\d)`)
  const printNamed = (name: string) =>
    screen.queryByRole("menuitem", { name: printed(name) })

  it("offers Other prints on a confident Scan, sending its one Card and fetching only once opened", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    reprintPool = [cardOf(11)]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    expect(reprintCalls).toEqual([])

    await openOtherPrints("Card 1")

    expect(
      await screen.findByRole("menuitem", { name: printed("Card 11") })
    ).toBeTruthy()
    expect(reprintCalls).toEqual(["?cardIds=card-1&offset=0"])
  })

  it("loads five at a time and drops Load more once no Reprints remain", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    reprintPool = [11, 12, 13, 14, 15, 16, 17].map(cardOf)
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await openOtherPrints("Card 1")

    expect(
      await screen.findByRole("menuitem", { name: printed("Card 15") })
    ).toBeTruthy()
    expect(printNamed("Card 16")).toBeNull()
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Load more" })
    )

    expect(
      await screen.findByRole("menuitem", { name: printed("Card 17") })
    ).toBeTruthy()
    expect(reprintCalls.at(-1)).toContain("offset=5")
    expect(screen.queryByRole("menuitem", { name: "Load more" })).toBeNull()
  })

  it("never offers a Card already among the Scan's candidates, nor one twice across pages", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    reprintPool = [11, 12, 13, 14, 15, 15, 16].map(cardOf)
    reprintPool.push(cardOf(2))
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await openOtherPrints("Card 1")

    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Load more" })
    )
    expect(
      await screen.findByRole("menuitem", { name: printed("Card 16") })
    ).toBeTruthy()
    expect(
      screen.getAllByRole("menuitem", { name: printed("Card 15") })
    ).toHaveLength(1)
    expect(printNamed("Card 2")).toBeNull()
  })

  it("says there are no other prints and offers nothing to pick", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await openOtherPrints("Card 1")

    expect(await screen.findByText("No other prints")).toBeTruthy()
    expect(screen.queryByRole("menuitem", { name: "Load more" })).toBeNull()
  })

  it("makes a picked Reprint the chosen Card, which can be changed again", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    reprintPool = [cardOf(11)]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await openOtherPrints("Card 1")
    await userEvent.click(
      await screen.findByRole("menuitem", { name: printed("Card 11") })
    )

    let row = await rowOf("Card 11")
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 11" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemradio", { name: printed("Card 1") })
    )

    // The menu may stay mounted while it closes, so the row is read by its list item.
    row = screen.getByRole("listitem")
    expect(within(row).getByText("Card 1")).toBeTruthy()
  })

  it("keeps a picked Reprint and its position across a reload", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    reprintPool = [11, 12, 13, 14, 15, 16, 17].map(cardOf)
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await openOtherPrints("Card 1")
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Load more" })
    )
    await userEvent.click(
      await screen.findByRole("menuitem", { name: printed("Card 16") })
    )
    await waitFor(async () =>
      expect(await loadDraft("user-1", "col-1")).toHaveLength(1)
    )

    expect(await loadDraft("user-1", "col-1")).toEqual([
      expect.objectContaining({ cardId: "card-16", reprintPosition: 5 }),
    ])
    cleanup()
    renderScreen()
    await expandSheet()
    const row = await rowOf("Card 16")
    expect(within(row).getByText("Card 16")).toBeTruthy()
  })

  it("shows a retry when the list fails, and leaves the Scan as it was", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    reprintPool = [cardOf(11)]
    reprintFailures = 1
    renderScreen()
    await capture()
    const row = await rowOf("Card 1")
    await openOtherPrints("Card 1")

    expect(await screen.findByRole("alert")).toBeTruthy()
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Retry" })
    )
    expect(
      await screen.findByRole("menuitem", { name: printed("Card 11") })
    ).toBeTruthy()
    expect(within(row).getByText("Card 1")).toBeTruthy()
  })
})

describe("a Card that will not load", () => {
  it("says so and retries the lookup", async () => {
    await saveDraft("user-1", "col-1", [
      { cardId: "card-2", quantity: 1, candidateIds: [] },
    ])
    failLookup = new Set(["card-2"])
    renderScreen()
    await expandSheet()

    const row = await rowOf("Could not load card")
    failLookup = new Set()
    await userEvent.click(
      within(row).getByRole("button", { name: "Retry loading scan" })
    )

    expect(await rowOf("Card 2")).toBeTruthy()
  })
})

describe("marking a Scan", () => {
  it("says no card was found for a server-flagged empty, with Retry, Remove and no card to pick", async () => {
    replies = [{ confident: false, candidates: [], empty: true }]
    renderScreen()
    await capture()

    // The sheet is collapsed, so its one row is the Scan. The status line says the same words.
    const row = await screen.findByRole("listitem")
    await waitFor(() =>
      expect(within(row).getByText("No card found.")).toBeTruthy()
    )
    expect(within(row).getByRole("button", { name: "Retry" })).toBeTruthy()
    expect(
      within(row).getByRole("button", { name: "Remove scan" })
    ).toBeTruthy()
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for scan" })
    )
    expect(
      await screen.findByRole("menuitemcheckbox", {
        name: "No card in the photo",
      })
    ).toBeTruthy()
    expect(screen.queryByRole("menuitemradio")).toBeNull()
    expect(screen.queryByRole("menuitem", { name: "Other prints" })).toBeNull()
  })

  it("marks a confident Scan Unidentified from the picker, worded for the photo, and keeps its card", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Card 1")
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemcheckbox", {
        name: "A card, but none of these",
      })
    )

    await waitFor(() =>
      expect(
        within(row).getByText("Marked: a card, but none of these")
      ).toBeTruthy()
    )
    expect(within(row).getByText("Card 1")).toBeTruthy()
    expect(screen.queryByText(/slot/i)).toBeNull()
  })

  it("changes a marked Scan back to one of its candidates from the picker", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Card 1")
    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    await userEvent.click(
      await screen.findByRole("menuitemcheckbox", {
        name: "No card in the photo",
      })
    )
    await waitFor(() =>
      expect(within(row).getByText("Marked: no card in the photo")).toBeTruthy()
    )

    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
    // The mark is the selected item, not the card it was on.
    expect(
      (
        await screen.findByRole("menuitemcheckbox", {
          name: "No card in the photo",
        })
      ).getAttribute("aria-checked")
    ).toBe("true")
    for (const radio of screen.getAllByRole("menuitemradio"))
      expect(radio.getAttribute("aria-checked")).toBe("false")
    await userEvent.click(
      await screen.findByRole("menuitemradio", { name: /Card 2/ })
    )
    await waitFor(() => expect(within(row).queryByText(/Marked:/)).toBeNull())
    expect(within(row).getByText("Card 2")).toBeTruthy()
  })
})

describe("without a camera", () => {
  it("explains the denial and still takes a picked photo", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    const picked = source({ camera: "denied" })
    renderScreen("col-1", picked)

    expect(screen.getByRole("alert").textContent).toMatch(/denied/)
    expect(screen.queryByRole("button", { name: "Capture" })).toBeNull()

    // SAFETY: the picker is the only file input on the screen.
    const input = document.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement
    await userEvent.upload(
      input,
      new File(["x"], "card.jpg", { type: "image/jpeg" })
    )

    expect(await rowOf("Card 1")).toBeTruthy()
  })
})
