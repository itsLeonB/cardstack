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
import type {
  CardSummary,
  MatchCandidate,
  ScanFeedbackRowInput,
} from "@/generated/models"
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
let feedbackBodies: { rows: ScanFeedbackRowInput[] }[] = []
let feedbackMode: "ok" | "fail" = "ok"
let uploadRequests: number[] = []
let bucketMode: "ok" | "unconfigured" = "ok"
let photoPuts: string[] = []
let putMode: "ok" | "refused" = "ok"
let collectionWrites: { cardId: string; quantity: number }[][] = []
let placementFails = false

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
  if (url.pathname === "/collections/col-1") {
    return Promise.resolve(
      json({
        data: {
          id: "col-1",
          title: "Binder",
          description: "",
          cardCount: 0,
          maxCardCount: 0,
          binder: null,
        },
      })
    )
  }
  if (url.pathname === "/collections/col-1/entries") {
    if (init?.method !== "PATCH")
      return Promise.resolve(
        json({ data: [], meta: { total: 0, page: 1, limit: 100 } })
      )
    // SAFETY: the only PATCH this fake receives is the bulk update's.
    const { items } = JSON.parse(String(init?.body)) as {
      items: { cardId: string; quantity: number }[]
    }
    if (placementFails)
      return Promise.resolve(json({ title: "boom", detail: "Try later." }, 503))
    collectionWrites.push(items)
    return Promise.resolve(
      json({
        data: items.map(({ cardId, quantity }) => ({
          cardId,
          quantity,
          status: "applied",
        })),
      })
    )
  }
  if (url.pathname === "/scan/photo-uploads") {
    // SAFETY: the only POST to this path is the upload-address request, whose body is { count }.
    const { count } = JSON.parse(String(init?.body)) as { count: number }
    uploadRequests.push(count)
    if (bucketMode === "unconfigured")
      return Promise.resolve(
        json(
          {
            title: "not configured",
            detail: "photo storage is not configured",
          },
          503
        )
      )
    const n = uploadRequests.length
    const addresses = Array.from({ length: count }, (_, i) => ({
      key: `profile-1/photo-${n}-${i}`,
      url: `https://photos.test/${n}/${i}`,
      expiresAt: "2026-10-21T00:05:00Z",
    }))
    return Promise.resolve(json({ data: { uploads: addresses } }))
  }
  if (url.pathname === "/scan/feedback") {
    // SAFETY: the only POST to this path is a feedback batch.
    feedbackBodies.push(
      JSON.parse(String(init?.body)) as { rows: ScanFeedbackRowInput[] }
    )
    return Promise.resolve(
      feedbackMode === "fail"
        ? json({ title: "boom" }, 500)
        : new Response(null, { status: 204 })
    )
  }
  // The presigned PUT goes to the bucket, not the API.
  if (url.host === "photos.test") {
    photoPuts.push(new Headers(init?.headers).get("Content-Type") ?? "")
    return Promise.resolve(
      new Response(null, { status: putMode === "refused" ? 403 : 200 })
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
  feedbackBodies = []
  feedbackMode = "ok"
  uploadRequests = []
  bucketMode = "ok"
  photoPuts = []
  putMode = "ok"
  collectionWrites = []
  placementFails = false
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
      screen.getByRole("menuitem", { name: "No card in the photo" })
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
      await screen.findByRole("menuitem", { name: "No card in the photo" })
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
      await screen.findByRole("menuitem", { name: "A card, but none of these" })
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
      screen.getByRole("menuitem", { name: "No card in the photo" })
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
      await screen.findByRole("menuitem", { name: "No card in the photo" })
    ).toBeTruthy()
    expect(screen.queryByRole("menuitemradio")).toBeNull()
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
      await screen.findByRole("menuitem", { name: "A card, but none of these" })
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
      await screen.findByRole("menuitem", { name: "No card in the photo" })
    )
    await waitFor(() =>
      expect(within(row).getByText("Marked: no card in the photo")).toBeTruthy()
    )

    await userEvent.click(
      within(row).getByRole("button", { name: "Change card for Card 1" })
    )
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

describe("recording card-scan feedback", () => {
  const settle = () => new Promise((resolve) => setTimeout(resolve, 20))
  const printed = (name: string) => new RegExp(`^${name}(?!\\d)`)

  const addToCollection = async () => {
    await userEvent.click(
      screen.getByRole("button", { name: "Review and add" })
    )
    await userEvent.click(
      await screen.findByRole("button", { name: "Add to Collection" })
    )
  }
  const chooseCard = async (from: string, to: string) => {
    const row = await rowOf(from)
    await userEvent.click(
      within(row).getByRole("button", { name: `Change card for ${from}` })
    )
    await userEvent.click(
      await screen.findByRole("menuitemradio", { name: printed(to) })
    )
  }
  const markFrom = async (name: string, label: string) => {
    await rowOf(name)
    await userEvent.click(
      screen.getByRole("button", { name: `Change card for ${name}` })
    )
    await userEvent.click(await screen.findByRole("menuitem", { name: label }))
  }

  it("records a confident match as a confirmation with its rank, and uploads no photo", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Card 1")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({
        source: "card_scan",
        collectionId: "col-1",
        kind: "confirmed",
        cardId: "card-1",
        rank: 1,
        rankSource: "top",
      }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("imageKey")
    expect(uploadRequests).toEqual([])
    await waitFor(async () =>
      expect(await loadDraft("user-1", "col-1")).toEqual([])
    )
  })

  it("records a switch to another candidate, with its rank and its photo's key", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    renderScreen()
    await capture()
    await chooseCard("Card 1", "Card 2")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({
        kind: "switched",
        cardId: "card-2",
        rank: 2,
        rankSource: "top",
        imageKey: expect.stringMatching(/^profile-1\//),
      }),
    ])
    expect(uploadRequests).toEqual([1])
    expect(photoPuts).toEqual(["image/jpeg"])
  })

  it("records a Reprint with its rank counted across the Reprints pages", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    reprintPool = [11, 12, 13, 14, 15, 16, 17].map(cardOf)
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await userEvent.click(
      screen.getByRole("button", { name: "Change card for Card 1" })
    )
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Other prints" })
    )
    await userEvent.click(
      await screen.findByRole("menuitem", { name: "Load more" })
    )
    await userEvent.click(
      await screen.findByRole("menuitem", { name: printed("Card 16") })
    )

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({
        kind: "reprint",
        cardId: "card-16",
        rank: 6,
        rankSource: "reprint",
        imageKey: expect.stringMatching(/^profile-1\//),
      }),
    ])
    expect(uploadRequests).toEqual([1])
  })

  it("records an Empty mark on a photo the matcher read as cards, with no Card and its photo", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    renderScreen()
    await capture()
    await markFrom("Card 1", "No card in the photo")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ source: "card_scan", kind: "empty" }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("cardId")
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("rank")
    expect(uploadRequests).toEqual([1])
  })

  it("records an Unidentified mark on a confident match as a correction with its photo", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await markFrom("Card 1", "A card, but none of these")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ kind: "unidentified" }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("cardId")
    expect(uploadRequests).toEqual([1])
  })

  it("records an empty photo the matcher flagged as a confirmation, with no photo", async () => {
    replies = [{ confident: false, candidates: [], empty: true }]
    renderScreen()
    await capture()
    await screen.findAllByText("No card found.")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ source: "card_scan", kind: "confirmed" }),
    ])
    expect(uploadRequests).toEqual([])
  })

  it("records only the Scans the add placed, not a Scan removed from the draft", async () => {
    replies = [
      { confident: true, candidates: [candidate(1)] },
      { confident: true, candidates: [candidate(2)] },
    ]
    renderScreen()
    await capture()
    await capture()
    const row = await rowOf("Card 2")
    fireEvent.click(within(row).getByRole("button", { name: /^Remove/ }))
    await rowOf("Card 1")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ kind: "confirmed", cardId: "card-1" }),
    ])
  })

  it("records nothing for a failed match, which the add leaves in the draft", async () => {
    replies = ["fail", { confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await capture()
    await rowOf("Card 1")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ kind: "confirmed", cardId: "card-1" }),
    ])
  })

  it("records nothing when the draft is discarded", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()
    await rowOf("Card 1")
    await expandSheet()

    await userEvent.click(screen.getByRole("button", { name: "Discard draft" }))
    await settle()

    expect(feedbackBodies).toEqual([])
  })

  it("records nothing and keeps the draft when the placement fails", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    placementFails = true
    renderScreen()
    await capture()
    await rowOf("Card 1")

    await addToCollection()
    await screen.findByRole("alert")
    await settle()

    expect(feedbackBodies).toEqual([])
    expect(uploadRequests).toEqual([])
  })

  it("leaves the add as it was when the photo bucket is not configured", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    bucketMode = "unconfigured"
    const onAdded = vi.fn()
    renderScreen("col-1", source(), onAdded)
    await capture()
    await chooseCard("Card 1", "Card 2")

    await addToCollection()

    await waitFor(() => expect(onAdded).toHaveBeenCalled())
    await settle()
    expect(collectionWrites).toEqual([[{ cardId: "card-2", quantity: 1 }]])
    expect(feedbackBodies).toEqual([])
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("records a Correction without its key when its photo fails to upload", async () => {
    replies = [{ confident: false, candidates: [candidate(1), candidate(2)] }]
    putMode = "refused"
    renderScreen()
    await capture()
    await chooseCard("Card 1", "Card 2")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ kind: "switched", cardId: "card-2" }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("imageKey")
  })

  it("leaves the add as it was when the feedback call fails, and clears the draft", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    feedbackMode = "fail"
    const onAdded = vi.fn()
    renderScreen("col-1", source(), onAdded)
    await capture()
    await rowOf("Card 1")

    await addToCollection()

    await waitFor(() => expect(onAdded).toHaveBeenCalled())
    await waitFor(async () =>
      expect(await loadDraft("user-1", "col-1")).toEqual([])
    )
    expect(collectionWrites).toEqual([[{ cardId: "card-1", quantity: 1 }]])
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("posts a Correction whose photo is gone with no key, and uploads nothing for it", async () => {
    // A restored Scan keeps its Detection, but its photo did not survive the reload.
    await saveDraft("user-1", "col-1", [
      {
        cardId: "card-2",
        quantity: 1,
        candidateIds: ["card-1", "card-2"],
        detection: {
          data: {
            candidates: [candidate(1), candidate(2)],
            confident: false,
            empty: false,
          },
        },
      },
    ])
    renderScreen()
    await expandSheet()
    await rowOf("Card 2")

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({
        kind: "switched",
        cardId: "card-2",
        rank: 2,
        rankSource: "top",
      }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("imageKey")
    expect(uploadRequests).toEqual([])
  })

  it("records Unidentified kept on a Detection with no candidates as a confirmation, with no key", async () => {
    await saveDraft("user-1", "col-1", [
      {
        quantity: 1,
        candidateIds: [],
        mark: "unidentified",
        photo: jpeg(),
        detection: {
          data: { candidates: [], confident: false, empty: false },
        },
      },
    ])
    renderScreen()

    await addToCollection()

    await waitFor(() => expect(feedbackBodies).toHaveLength(1))
    expect(feedbackBodies[0].rows).toEqual([
      expect.objectContaining({ source: "card_scan", kind: "confirmed" }),
    ])
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("cardId")
    expect(feedbackBodies[0].rows[0]).not.toHaveProperty("imageKey")
    expect(uploadRequests).toEqual([])
  })
})
