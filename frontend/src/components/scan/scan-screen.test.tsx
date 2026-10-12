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

type Reply = { confident: boolean; candidates: MatchCandidate[] } | "fail"

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
    if (init?.body instanceof Blob) uploads.push(init.body)
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
  return Promise.resolve(json({ title: "unexpected" }, 500))
}

// Answers the oldest held match.
const answer = (reply: Reply) => waiting.shift()!.answer(reply)

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

function renderScreen(
  collectionId = "col-1",
  frames = source(),
  onAdded = () => {}
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanScreen
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
  vi.stubGlobal("fetch", vi.fn(fakeApi))
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
    expect(uploads[0]).toBeInstanceOf(Blob)
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

  it("shows a confident match as its card, with no dropdown", async () => {
    replies = [{ confident: true, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    const row = await rowOf("Card 1")
    expect(within(row).getByText(/Starter Set · No\. 1/)).toBeTruthy()
    expect(screen.queryByRole("button", { name: /Change card for/ })).toBeNull()
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
    replies = ["fail"]
    renderScreen()
    await capture()
    await rowOf("Could not match this photo.")

    expect(
      screen
        .getByRole("button", { name: "Review and add" })
        .hasAttribute("disabled")
    ).toBe(true)
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
    expect(localStorage.getItem("cardstack:draft-addition:col-1")).toBeNull()
  })
})

describe("persistence", () => {
  it("restores resolved Scans with their alternatives after a reload", async () => {
    saveDraft("col-1", [
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

    expect(loadDraft("col-1")).toEqual([])
  })

  it("keeps each Collection's draft separate", () => {
    saveDraft("col-2", [{ cardId: "card-9", quantity: 1, candidateIds: [] }])
    renderScreen("col-1")

    expect(screen.getByText("Scanned cards will appear here.")).toBeTruthy()
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

describe("the candidates", () => {
  it("shows no dropdown when the matcher offers a single candidate", async () => {
    replies = [{ confident: false, candidates: [candidate(1)] }]
    renderScreen()
    await capture()

    await rowOf("Card 1")
    expect(screen.queryByRole("button", { name: /Change card for/ })).toBeNull()
  })

  it("shows the picked card at once, before the sheet's own lookup of it answers", async () => {
    saveDraft("col-1", [
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

describe("a Card that will not load", () => {
  it("says so and retries the lookup", async () => {
    saveDraft("col-1", [{ cardId: "card-2", quantity: 1, candidateIds: [] }])
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
