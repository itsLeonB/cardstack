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
import {
  RouterProvider,
  createBrowserHistory,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
import { ScanPageFlow } from "./page-scan"
import {
  loadPageReview,
  renamePageReview,
  savePageReview,
} from "@/lib/page-review-store"
import type { PendingPageReview } from "@/lib/page-review-store"
import type { PageCell } from "@/lib/use-page-scan"
import type { FrameSource } from "@/lib/frame-source"
import { frameLongSide } from "@/lib/page-cut"
import type { Corners, Grid } from "@/lib/page-cut"
import type {
  BinderPage,
  BinderPageReplaceBody,
  BinderSlot,
  EnvelopeBinderPageReplaceResult,
  CardSummary,
  MatchCandidate,
  MatchResult,
} from "@/generated/models"

// Only the network is faked (`fetch`) and the camera and canvas are faked:
// the frame source hands over a photo, and the cut returns one JPEG per cell.
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

// Every match waits until the test answers it, oldest first.
let held: ((response: Response) => void)[] = []
let matchCalls = 0
// Each Replace Page the flow sends, and the grid indexes the server reports as unknown Cards.
let saves: BinderPageReplaceBody[] = []
let unknownIndexes: number[] = []

type Body =
  { data: MatchResult } | EnvelopeBinderPageReplaceResult | { title: string }
const json = (body: Body, status = 200) =>
  new Response(JSON.stringify(body), { status })

// The 2 by 2 test Page: grid index 1 is row 1, column 2.
function savedPage(
  body: BinderPageReplaceBody
): EnvelopeBinderPageReplaceResult {
  const slots = (body.slots ?? []).map((slot, index) => {
    const unknown = unknownIndexes.includes(index)
    return {
      row: Math.floor(index / 2) + 1,
      column: (index % 2) + 1,
      state: unknown ? "unidentified" : slot.state,
      ...(unknown && { reason: "unknown_card" }),
    }
  })
  return { data: { slots } }
}

function fakeApi(input: RequestInfo | URL, init?: RequestInit) {
  const url = new URL(String(input))
  if (url.pathname === "/scan/match") {
    matchCalls++
    return new Promise<Response>((resolve) => held.push(resolve))
  }
  if (
    url.pathname === "/collections/col-1/binder/pages/1" &&
    init?.method === "PUT"
  ) {
    const body: BinderPageReplaceBody = JSON.parse(String(init.body))
    saves.push(body)
    return Promise.resolve(json(savedPage(body)))
  }
  return Promise.resolve(json({ title: "unexpected" }, 500))
}

/** Answers the held match at `index`, for the cases where a match does not answer oldest first. */
function answerAt(index: number, reply: Reply) {
  const [resolve] = held.splice(index, 1)
  resolve(
    reply === "fail" ? json({ title: "boom" }, 503) : json({ data: reply })
  )
}

/** Answers the oldest held match. */
function answer(reply: Reply) {
  answerAt(0, reply)
}

const emptySlot = (row: number, column: number) =>
  // SAFETY: an empty Slot has no card, and the screen reads `card` only for a Slot in the card state.
  ({ row, column, state: "empty" }) as BinderSlot

const page: BinderPage = {
  page: 1,
  leaf: 1,
  side: "front",
  layout: { rows: 2, columns: 2, leafCount: 1, capacity: 4 },
  cardTotal: 0,
  unidentifiedCount: 0,
  slots: [emptySlot(1, 1), emptySlot(1, 2), emptySlot(2, 1), emptySlot(2, 2)],
}

const jpeg = () => new Blob(["jpeg"], { type: "image/jpeg" })
function source(overrides: Partial<FrameSource> = {}): FrameSource {
  return {
    camera: "unavailable",
    videoRef: () => {},
    captureCamera: async () => jpeg(),
    fromFile: async () => jpeg(),
    captureFrame: async () => jpeg(),
    frameFromFile: async () => jpeg(),
    ...overrides,
  }
}

type CutPhoto = (frame: Blob, corners: Corners, grid: Grid) => Promise<Blob[]>

// One JPEG per cell, row by row, so each cell's photo is distinct.
const cut = vi.fn<CutPhoto>(async (_frame, _corners, grid) =>
  Array.from(
    { length: grid.rows * grid.columns },
    (_, i) => new Blob([`cell-${i}`], { type: "image/jpeg" })
  )
)

// The flow sits on a route, as it does in the app, so the leave blocker has a router to guard.
function renderFlow(
  frames = source(),
  cutPhoto: CutPhoto = cut,
  onClose = vi.fn(),
  current = page,
  history = createMemoryHistory({ initialEntries: ["/"] }),
  resume?: PendingPageReview
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const root = createRootRoute()
  const flow = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: () => (
      <ScanPageFlow
        collectionId="col-1"
        page={current}
        resume={resume}
        userId="user-1"
        source={frames}
        cut={cutPhoto}
        onClose={onClose}
      />
    ),
  })
  const elsewhere = createRoute({
    getParentRoute: () => root,
    path: "/elsewhere",
    component: () => <p>Somewhere else</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([flow, elsewhere]),
    history,
  })
  const view = render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return { ...view, router }
}

/** Picks a photo and cuts it, leaving the review with all four cells matching. */
async function scanPhoto(user: ReturnType<typeof userEvent.setup>) {
  await user.upload(
    await screen.findByLabelText("Choose a photo"),
    new File(["photo"], "page.jpg", { type: "image/jpeg" })
  )
  await user.click(await screen.findByRole("button", { name: "Cut and match" }))
}

/** Answers the first three matches in order, then the fourth once the queue starts it. */
async function answerAll(replies: Reply[]) {
  answer(replies[0])
  await waitFor(() => expect(matchCalls).toBe(4))
  replies.slice(1).forEach(answer)
}

const cell = (row: number, column: number) =>
  screen.getByRole("group", { name: `Row ${row}, column ${column}` })

const confirmButton = () => screen.getByRole("button", { name: "Confirm" })

// A cell's pill: its button named by the cell's position, or null while matching.
const pill = (row: number, column: number) =>
  within(cell(row, column)).queryByRole("button", {
    name: new RegExp(`^Row ${row}, column ${column}:`),
  })

const openMenu = async (
  user: ReturnType<typeof userEvent.setup>,
  row: number,
  column: number
) => {
  await user.click(
    await within(cell(row, column)).findByRole("button", {
      name: new RegExp(`^Row ${row}, column ${column}:`),
    })
  )
  return screen.findByRole("menu")
}

/** Opens a cell's menu and picks the item whose name matches. */
async function pick(
  user: ReturnType<typeof userEvent.setup>,
  row: number,
  column: number,
  name: string | RegExp
) {
  await openMenu(user, row, column)
  await user.click(await screen.findByRole("menuitemradio", { name }))
  await waitFor(() => expect(screen.queryByRole("menu")).toBeNull())
}

// Four cells, each a confident match for its own card.
const fourConfident: Reply[] = [1, 2, 3, 4].map((n) => ({
  confident: true,
  candidates: [candidate(n)],
}))

beforeEach(() => {
  held = []
  matchCalls = 0
  saves = []
  unknownIndexes = []
  cut.mockClear()
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
  vi.stubGlobal("fetch", vi.fn(fakeApi))
  URL.createObjectURL = vi.fn(() => "blob:photo")
  URL.revokeObjectURL = vi.fn()
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("ScanPageFlow", () => {
  it("cuts the photo into the Page's cells and keeps three matches in flight", async () => {
    const user = userEvent.setup()
    renderFlow()

    await scanPhoto(user)

    await waitFor(() => expect(matchCalls).toBe(3))
    expect(cut.mock.calls[0][2]).toEqual({ rows: 2, columns: 2 })
    expect(cut.mock.calls[0][1][0]).toEqual({ x: 0.15, y: 0.1 })
    answer({ confident: true, candidates: [candidate(1)] })
    await waitFor(() => expect(matchCalls).toBe(4))
  })

  it("nudges a corner from the keyboard before cutting", async () => {
    const user = userEvent.setup()
    renderFlow()
    await user.upload(
      await screen.findByLabelText("Choose a photo"),
      new File(["photo"], "page.jpg", { type: "image/jpeg" })
    )

    const topLeft = await screen.findByRole("button", {
      name: "Top left corner",
    })
    topLeft.focus()
    await user.keyboard("{ArrowLeft}")
    await user.click(screen.getByRole("button", { name: "Cut and match" }))

    await waitFor(() => expect(cut).toHaveBeenCalled())
    expect(cut.mock.calls[0][1][0].x).toBeCloseTo(0.14)
  })

  it("shows each cell's pill: the top candidate, no good candidate, failed, never unsure", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))
    // While matching the pill has no menu.
    expect(within(cell(1, 1)).getByText("Matching…")).toBeTruthy()
    expect(pill(1, 1)).toBeNull()

    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      { confident: false, candidates: [candidate(2), candidate(3)] },
      { confident: false, candidates: [] },
      "fail",
    ])

    // A confident cell shows its Card's name, then set code and local ID.
    const first = await within(cell(1, 1)).findByRole("button", {
      name: /^Row 1, column 1: Card 1, Starter Set 1/,
    })
    expect(first.textContent).toBe("Card 1SCE · #1")
    // An unsure match starts on its top candidate.
    expect(pill(1, 2)?.textContent).toBe("Card 2SCE · #2")
    expect(await within(cell(2, 1)).findByText("Unidentified")).toBeTruthy()
    expect(pill(2, 2)?.textContent).toBe("Couldn't match")
    expect(
      within(cell(2, 2)).getByRole("button", { name: "Retry" })
    ).toBeTruthy()
    expect(document.body.textContent).not.toMatch(/unsure/i)
  })

  it("opens a menu of the candidates and of Empty and Unidentified, one choice marked", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      { confident: false, candidates: [candidate(2), candidate(3)] },
      { confident: false, candidates: [] },
      "fail",
    ])

    const items = async (row: number, column: number) => {
      const menu = await openMenu(user, row, column)
      const rows = within(menu)
        .getAllByRole("menuitemradio")
        .map((item) => [item.textContent, item.getAttribute("aria-checked")])
      await user.keyboard("{Escape}")
      await waitFor(() => expect(screen.queryByRole("menu")).toBeNull())
      return rows
    }
    // A confident cell's menu holds its one candidate, Empty and Unidentified.
    expect(await items(1, 1)).toEqual([
      ["Card 1SCE · #1", "true"],
      ["Empty", "false"],
      ["Unidentified", "false"],
    ])
    expect(await items(1, 2)).toEqual([
      ["Card 2SCE · #2", "true"],
      ["Card 3SCE · #3", "false"],
      ["Empty", "false"],
      ["Unidentified", "false"],
    ])
    // With no candidate, or a failed match, the cell is Unidentified.
    expect(await items(2, 1)).toEqual([
      ["Empty", "false"],
      ["Unidentified", "true"],
    ])
    expect(await items(2, 2)).toEqual([
      ["Empty", "false"],
      ["Unidentified", "true"],
    ])
  })

  it("is a single choice: Empty and Unidentified keep the candidates, a candidate clears them", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await answerAll([
      { confident: false, candidates: [candidate(1), candidate(2)] },
      ...fourConfident.slice(1),
    ])
    const chosen = () => pill(1, 1)?.textContent

    await pick(user, 1, 1, /^Card 2/)
    expect(chosen()).toBe("Card 2SCE · #2")
    await pick(user, 1, 1, "Empty")
    expect(chosen()).toBe("Empty")
    await pick(user, 1, 1, "Unidentified")
    expect(chosen()).toBe("Unidentified")
    // The candidates stay, so picking one restores it and clears the mark.
    await pick(user, 1, 1, /^Card 2/)
    expect(chosen()).toBe("Card 2SCE · #2")
    // Picking the top candidate returns to the matcher's answer.
    await pick(user, 1, 1, /^Card 1/)
    expect(chosen()).toBe("Card 1SCE · #1")
  })

  it("retries a failed cell, which goes back to the queue", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))
    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      "fail",
      { confident: true, candidates: [candidate(2)] },
      { confident: true, candidates: [candidate(3)] },
    ])

    await user.click(
      await within(cell(1, 2)).findByRole("button", { name: "Retry" })
    )
    await waitFor(() => expect(matchCalls).toBe(5))
    answer({ confident: true, candidates: [candidate(4)] })

    expect(await within(cell(1, 2)).findByText("Card 4")).toBeTruthy()
  })

  it("keeps the photo and says so when the cut fails", async () => {
    const user = userEvent.setup()
    const failing = vi.fn<CutPhoto>(async () => {
      throw new Error("cut failed")
    })
    renderFlow(source(), failing)
    await scanPhoto(user)

    expect(
      await screen.findByText(
        "Could not cut this photo. Move the corners and try again."
      )
    ).toBeTruthy()
    expect(
      screen
        .getByRole("button", { name: "Cut and match" })
        .hasAttribute("disabled")
    ).toBe(false)
  })

  it("keeps Confirm disabled while a cell is still matching", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)

    expect(confirmButton().hasAttribute("disabled")).toBe(true)
    await answerAll(fourConfident)
    await waitFor(() =>
      expect(confirmButton().hasAttribute("disabled")).toBe(false)
    )
  })

  it("saves each cell as the scan found it, with the menu's choices overriding it", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderFlow(source(), cut, onClose)
    await scanPhoto(user)
    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      "fail",
      { confident: false, candidates: [] },
      { confident: false, candidates: [candidate(2), candidate(3)] },
    ])

    await pick(user, 1, 1, "Empty")
    await pick(user, 2, 2, /^Card 3/)
    await user.click(confirmButton())

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(saves).toEqual([
      {
        slots: [
          { state: "empty" },
          { state: "unidentified" },
          { state: "unidentified" },
          { state: "card", cardId: "card-3" },
        ],
      },
    ])
  })

  it("highlights a cell whose state would change the Slot the Page holds", async () => {
    const user = userEvent.setup()
    const holding: BinderPage = {
      ...page,
      cardTotal: 1,
      slots: [
        { row: 1, column: 1, state: "card", card: cardOf(1) },
        emptySlot(1, 2),
        emptySlot(2, 1),
        emptySlot(2, 2),
      ],
    }
    renderFlow(source(), cut, vi.fn(), holding)
    await scanPhoto(user)
    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      "fail",
      { confident: true, candidates: [candidate(3)] },
      { confident: true, candidates: [candidate(4)] },
    ])

    const image = (row: number, column: number) =>
      within(cell(row, column)).getByRole("img")
    await waitFor(() =>
      expect(pill(1, 2)?.getAttribute("aria-label")).toBe(
        "Row 1, column 2: Couldn't match, Unidentified. Changes this Slot"
      )
    )
    expect(image(1, 2).getAttribute("data-changed")).toBe("true")
    expect(pill(1, 1)?.getAttribute("aria-label")).toBe(
      "Row 1, column 1: Card 1, Starter Set 1"
    )
    expect(image(1, 1).getAttribute("data-changed")).toBeNull()
    expect(document.body.textContent).not.toContain("Changes this Slot")
  })

  it("names a Slot the server stored as Unidentified for an unknown Card, then returns", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    unknownIndexes = [1]
    renderFlow(source(), cut, onClose)
    await scanPhoto(user)
    await answerAll(fourConfident)

    await user.click(confirmButton())

    expect(await screen.findByText("Row 1, column 2")).toBeTruthy()
    expect(onClose).not.toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "Done" }))
    expect(onClose).toHaveBeenCalled()
  })

  it("keeps Confirm disabled while another cell is still matching, then saves the mark", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))

    // Cell 1 is answered and marked Empty while the others are still loading.
    answerAt(0, fourConfident[0])
    await waitFor(() => expect(matchCalls).toBe(4))
    await pick(user, 1, 1, "Empty")
    expect(confirmButton().hasAttribute("disabled")).toBe(true)

    answerAt(0, fourConfident[1])
    answerAt(0, fourConfident[2])
    answerAt(0, fourConfident[3])
    await waitFor(() =>
      expect(confirmButton().hasAttribute("disabled")).toBe(false)
    )
    await user.click(confirmButton())
    await waitFor(() => expect(saves).toHaveLength(1))
    expect(saves[0].slots).toEqual([
      { state: "empty" },
      { state: "card", cardId: "card-2" },
      { state: "card", cardId: "card-3" },
      { state: "card", cardId: "card-4" },
    ])
  })

  it("refuses corners that cross, and cuts once they are fixed", async () => {
    const user = userEvent.setup()
    renderFlow()
    await user.upload(
      await screen.findByLabelText("Choose a photo"),
      new File(["photo"], "page.jpg", { type: "image/jpeg" })
    )
    const topLeft = await screen.findByRole("button", {
      name: "Top left corner",
    })

    // Each Shift arrow moves a corner 0.05; sixteen carry it past the top right corner.
    topLeft.focus()
    for (let i = 0; i < 16; i++)
      await user.keyboard("{Shift>}{ArrowRight}{/Shift}")
    expect(
      await screen.findByText(
        /Move the corners so they make a four-sided shape/
      )
    ).toBeTruthy()
    expect(
      screen
        .getByRole("button", { name: "Cut and match" })
        .hasAttribute("disabled")
    ).toBe(true)

    for (let i = 0; i < 16; i++)
      await user.keyboard("{Shift>}{ArrowLeft}{/Shift}")
    await waitFor(() =>
      expect(
        screen
          .getByRole("button", { name: "Cut and match" })
          .hasAttribute("disabled")
      ).toBe(false)
    )
    await user.click(screen.getByRole("button", { name: "Cut and match" }))
    expect(cut).toHaveBeenCalledTimes(1)
  })

  it("ignores matches from a cut that was replaced", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))

    await user.click(screen.getByRole("button", { name: "Adjust corners" }))
    await user.click(
      await screen.findByRole("button", { name: "Cut and match" })
    )
    await waitFor(() => expect(matchCalls).toBe(6))

    // The first cut's three matches answer now, with a Card the new review must not show.
    answer({ confident: true, candidates: [candidate(9)] })
    answer({ confident: true, candidates: [candidate(9)] })
    answer({ confident: true, candidates: [candidate(9)] })
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(screen.queryByText("Card 9")).toBeNull()
    expect(within(cell(1, 1)).getByText("Matching…")).toBeTruthy()

    answer(fourConfident[0])
    await waitFor(() => expect(matchCalls).toBe(7))
    fourConfident.slice(1).forEach(answer)
    await user.click(await screen.findByRole("button", { name: "Confirm" }))
    await waitFor(() => expect(saves).toHaveLength(1))
    expect(saves[0].slots?.map((slot) => slot.cardId)).toEqual([
      "card-1",
      "card-2",
      "card-3",
      "card-4",
    ])
  })

  it("drops a match that returns after leaving, and starts no queued cell", async () => {
    const user = userEvent.setup()
    const view = renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))

    view.unmount()
    answer(fourConfident[0])
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(matchCalls).toBe(3)
  })

  it("asks before a route change while the review has edits", async () => {
    // Without IndexedDB the review is memory-only, so the leave guard is on.
    vi.stubGlobal("indexedDB", undefined)
    const user = userEvent.setup()
    const { router } = renderFlow()
    await scanPhoto(user)
    await answerAll(fourConfident)
    await pick(user, 1, 1, "Empty")

    void router.history.push("/elsewhere")
    expect(await screen.findByText("Leave this review?")).toBeTruthy()
    expect(screen.queryByText("Somewhere else")).toBeNull()

    await user.click(screen.getByRole("button", { name: "Keep editing" }))
    await waitFor(() =>
      expect(screen.queryByText("Leave this review?")).toBeNull()
    )
    expect(screen.getByRole("heading", { name: "Scan Page 1" })).toBeTruthy()

    void router.history.push("/elsewhere")
    await user.click(await screen.findByRole("button", { name: "Leave" }))
    expect(await screen.findByText("Somewhere else")).toBeTruthy()
  })

  it("warns before the tab closes only while the review has edits", async () => {
    // Without IndexedDB the review is memory-only, so the leave guard is on.
    vi.stubGlobal("indexedDB", undefined)
    const user = userEvent.setup()
    // The browser history is the one that listens for the tab closing.
    renderFlow(source(), cut, vi.fn(), page, createBrowserHistory())
    await scanPhoto(user)
    await answerAll(fourConfident)

    const idle = new Event("beforeunload", { cancelable: true })
    window.dispatchEvent(idle)
    expect(idle.defaultPrevented).toBe(false)

    await pick(user, 1, 1, "Empty")
    const edited = new Event("beforeunload", { cancelable: true })
    window.dispatchEvent(edited)
    expect(edited.defaultPrevented).toBe(true)
  })

  it("asks before leaving a review that has edits", async () => {
    // Without IndexedDB the review is memory-only, so the leave guard is on.
    vi.stubGlobal("indexedDB", undefined)
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderFlow(source(), cut, onClose)
    await scanPhoto(user)
    await answerAll(fourConfident)
    await pick(user, 1, 1, "Empty")

    await user.click(screen.getByRole("button", { name: "Back to binder" }))
    expect(await screen.findByText("Leave this review?")).toBeTruthy()
    expect(onClose).not.toHaveBeenCalled()

    await user.click(screen.getByRole("button", { name: "Keep editing" }))
    await waitFor(() =>
      expect(screen.queryByText("Leave this review?")).toBeNull()
    )
    await user.click(screen.getByRole("button", { name: "Back to binder" }))
    await user.click(await screen.findByRole("button", { name: "Leave" }))
    expect(onClose).toHaveBeenCalled()
  })

  it("says the review is not saved, and guards leaving, when IndexedDB is missing", async () => {
    vi.stubGlobal("indexedDB", undefined)
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)

    expect(
      await screen.findByText(/This review is not being saved/)
    ).toBeTruthy()
  })
})

describe("ScanPageFlow, pending Page Review", () => {
  const key = { userId: "user-1", collectionId: "col-1", page: 1 }
  const reviewOf = (cells: PageCell[]): PendingPageReview => ({
    ...key,
    grid: { rows: 2, columns: 2 },
    cells,
  })
  const matched = (n: number): PageCell => ({
    photo: jpeg(),
    match: { status: "matched", confident: true, candidates: [cardOf(n)] },
    cardId: `card-${n}`,
  })
  const fourMatched = reviewOf([1, 2, 3, 4].map(matched))

  // Opens the flow on a saved review, as the binder does when the user resumes it.
  const resumeFlow = (saved: PendingPageReview, onClose = vi.fn()) =>
    renderFlow(
      source(),
      cut,
      onClose,
      page,
      createMemoryHistory({ initialEntries: ["/"] }),
      saved
    )

  it("saves the review after the cut, after each match, pick and mark", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(async () =>
      expect((await loadPageReview(key))?.cells).toHaveLength(4)
    )

    await answerAll(fourConfident)
    await waitFor(async () =>
      expect(
        (await loadPageReview(key))?.cells.every(
          (c) => c.match.status === "matched"
        )
      ).toBe(true)
    )

    await pick(user, 1, 1, "Empty")
    await waitFor(async () =>
      expect((await loadPageReview(key))?.cells[0].mark).toBe("empty")
    )
  })

  it("resumes a saved review, re-running only the cells that had not finished", async () => {
    const saved = reviewOf([
      matched(1),
      { photo: jpeg(), match: { status: "queued" } },
      { photo: jpeg(), match: { status: "loading" } },
      { photo: jpeg(), match: { status: "failed" } },
    ])
    resumeFlow(saved)

    await waitFor(() => expect(matchCalls).toBe(2))
    expect(pill(1, 1)?.textContent).toContain("Card 1")
    expect(
      within(cell(2, 2)).getByRole("button", { name: "Retry" })
    ).toBeTruthy()
    answer(fourConfident[1])
    answer(fourConfident[2])

    await waitFor(() => expect(pill(1, 2)?.textContent).toContain("Card 2"))
    expect(pill(2, 1)?.textContent).toContain("Card 3")
    expect(matchCalls).toBe(2)
  })

  it("offers Scan again in place of Adjust corners, which starts from the photo step", async () => {
    const user = userEvent.setup()
    resumeFlow(fourMatched)

    expect(screen.queryByRole("button", { name: "Adjust corners" })).toBeNull()
    await user.click(await screen.findByRole("button", { name: "Scan again" }))

    expect(await screen.findByLabelText("Choose a photo")).toBeTruthy()
    expect(matchCalls).toBe(0)
  })

  it("keeps the review through unmounting and remounting the screen", async () => {
    const user = userEvent.setup()
    const view = renderFlow()
    await scanPhoto(user)
    await answerAll(fourConfident)
    await pick(user, 1, 1, "Empty")
    await waitFor(async () =>
      expect((await loadPageReview(key))?.cells[0].mark).toBe("empty")
    )
    view.unmount()

    resumeFlow((await loadPageReview(key))!)
    expect(await screen.findByText("Check each cell.")).toBeTruthy()
    expect(matchCalls).toBe(4)
    expect(confirmButton().hasAttribute("disabled")).toBe(false)
  })

  it("leaves at once with edits in a saved review, with no dialog", async () => {
    const user = userEvent.setup()
    const { router } = resumeFlow(fourMatched)
    await screen.findByRole("group", { name: "Row 1, column 1" })
    await pick(user, 1, 1, "Empty")

    void router.history.push("/elsewhere")
    expect(await screen.findByText("Somewhere else")).toBeTruthy()
    expect(screen.queryByText("Leave this review?")).toBeNull()
  })

  it("clears the saved review when Confirm succeeds, and keeps it when Confirm fails", async () => {
    const user = userEvent.setup()
    await savePageReview(fourMatched)
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL, init?: RequestInit) =>
        init?.method === "PUT"
          ? Promise.resolve(json({ title: "boom" }, 500))
          : fakeApi(input, init)
      )
    )
    const failed = resumeFlow(fourMatched)
    await user.click(await screen.findByRole("button", { name: "Confirm" }))
    expect(await screen.findByRole("alert")).toBeTruthy()
    expect(await loadPageReview(key)).toBeDefined()
    failed.unmount()

    vi.stubGlobal("fetch", vi.fn(fakeApi))
    const onClose = vi.fn()
    resumeFlow(fourMatched, onClose)
    await user.click(await screen.findByRole("button", { name: "Confirm" }))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(await loadPageReview(key)).toBeUndefined()
  })
})

describe("ScanPageFlow, Page photo in a pending review", () => {
  const key = { userId: "user-1", collectionId: "col-1", page: 1 }
  const grid = { rows: 2, columns: 2 }
  const corners: Corners = [
    { x: 0.15, y: 0.1 },
    { x: 0.85, y: 0.1 },
    { x: 0.85, y: 0.9 },
    { x: 0.15, y: 0.9 },
  ]
  // The frame source hands back the downscaled photo; the raw file it came from is larger.
  // SAFETY: Node's Blob reads the same as the DOM Blob a cell's photo is typed as, and is what IndexedDB clones here (see page-review-store.test.ts).
  const photoOf = (text: string) => new NodeBlob([text]) as Blob
  const rawFile = () =>
    new File(["raw camera file, much larger than the photo"], "page.jpg", {
      type: "image/jpeg",
    })
  const photoText = async (review: PendingPageReview | undefined) =>
    review?.photo?.text()
  const matched = (n: number): PageCell => ({
    photo: jpeg(),
    match: { status: "matched", confident: true, candidates: [cardOf(n)] },
    cardId: `card-${n}`,
  })
  const savedWithPhoto = (text: string): PendingPageReview => ({
    userId: "user-1",
    collectionId: "col-1",
    page: 1,
    grid,
    cells: [1, 2, 3, 4].map(matched),
    photo: photoOf(text),
    corners,
  })
  const resumeWith = (
    saved: PendingPageReview | undefined,
    onClose = vi.fn()
  ) =>
    renderFlow(
      source(),
      cut,
      onClose,
      page,
      createMemoryHistory({ initialEntries: ["/"] }),
      saved
    )

  it("saves the downscaled Page photo, the grid and the corners with the cells", async () => {
    const user = userEvent.setup()
    const frameFromFile = vi.fn<FrameSource["frameFromFile"]>(async () =>
      photoOf("downscaled")
    )
    const file = rawFile()
    renderFlow(source({ frameFromFile }))
    await user.upload(await screen.findByLabelText("Choose a photo"), file)
    await user.click(
      await screen.findByRole("button", { name: "Cut and match" })
    )

    await waitFor(async () =>
      expect((await loadPageReview(key))?.cells).toHaveLength(4)
    )
    const saved = await loadPageReview(key)
    expect(frameFromFile.mock.calls[0][0]).toBe(file)
    expect(frameFromFile.mock.calls[0][1]).toBe(frameLongSide(grid))
    expect(await photoText(saved)).toBe("downscaled")
    expect(saved?.grid).toEqual(grid)
    expect(saved?.corners).toEqual(cut.mock.calls[0][1])
  })

  it("keeps the photo, grid and corners through leaving and resuming", async () => {
    const user = userEvent.setup()
    const view = renderFlow(
      source({ frameFromFile: async () => photoOf("downscaled") })
    )
    await user.upload(await screen.findByLabelText("Choose a photo"), rawFile())
    const topLeft = await screen.findByRole("button", {
      name: "Top left corner",
    })
    topLeft.focus()
    await user.keyboard("{ArrowLeft}")
    await user.click(screen.getByRole("button", { name: "Cut and match" }))
    await answerAll(fourConfident)
    await waitFor(async () =>
      expect(
        (await loadPageReview(key))?.cells.every(
          (c) => c.match.status === "matched"
        )
      ).toBe(true)
    )
    const cornersCut = cut.mock.calls[0][1]
    view.unmount()

    resumeWith(await loadPageReview(key))
    expect(await screen.findByText("Check each cell.")).toBeTruthy()
    await pick(user, 1, 1, "Empty")
    await waitFor(async () =>
      expect((await loadPageReview(key))?.cells[0].mark).toBe("empty")
    )

    const saved = await loadPageReview(key)
    expect(await photoText(saved)).toBe("downscaled")
    expect(saved?.grid).toEqual(grid)
    expect(saved?.corners).toEqual(cornersCut)
    expect(saved?.corners?.[0].x).toBeCloseTo(0.14)
  })

  it("resumes a review saved before the photo was kept, with no photo and no error", async () => {
    await savePageReview({
      userId: "user-1",
      collectionId: "col-1",
      page: 1,
      grid,
      cells: [1, 2, 3, 4].map(matched),
    })
    resumeWith(await loadPageReview(key))

    expect(await screen.findByText("Check each cell.")).toBeTruthy()
    expect(screen.queryByRole("alert")).toBeNull()
    expect(screen.getByRole("button", { name: "Scan again" })).toBeTruthy()
  })

  it("replaces the saved photo when Scan again cuts a new one", async () => {
    const user = userEvent.setup()
    await savePageReview(savedWithPhoto("old"))
    renderFlow(
      source({ frameFromFile: async () => photoOf("new") }),
      cut,
      vi.fn(),
      page,
      createMemoryHistory({ initialEntries: ["/"] }),
      await loadPageReview(key)
    )

    await user.click(await screen.findByRole("button", { name: "Scan again" }))
    await user.upload(await screen.findByLabelText("Choose a photo"), rawFile())
    await user.click(
      await screen.findByRole("button", { name: "Cut and match" })
    )

    await waitFor(async () =>
      expect(await photoText(await loadPageReview(key))).toBe("new")
    )
  })

  it("keeps the photo when Scan again is started but no new photo is cut", async () => {
    const user = userEvent.setup()
    await savePageReview(savedWithPhoto("old"))
    resumeWith(await loadPageReview(key))

    await user.click(await screen.findByRole("button", { name: "Scan again" }))
    await user.upload(await screen.findByLabelText("Choose a photo"), rawFile())
    await screen.findByRole("button", { name: "Cut and match" })
    await user.click(screen.getByRole("button", { name: "Back to binder" }))

    expect(await photoText(await loadPageReview(key))).toBe("old")
  })

  it("removes the photo with the record when Confirm succeeds", async () => {
    const user = userEvent.setup()
    await savePageReview(savedWithPhoto("photo"))
    const onClose = vi.fn()
    resumeWith(await loadPageReview(key), onClose)

    await user.click(await screen.findByRole("button", { name: "Confirm" }))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(await loadPageReview(key)).toBeUndefined()
  })

  it("keeps the photo when the review is moved to another Page", async () => {
    const user = userEvent.setup()
    await savePageReview(savedWithPhoto("moved"))
    await renamePageReview(key, 3)
    const moved = { ...key, page: 3 }
    renderFlow(
      source(),
      cut,
      vi.fn(),
      { ...page, page: 3 },
      createMemoryHistory({ initialEntries: ["/"] }),
      await loadPageReview(moved)
    )

    await screen.findByText("Check each cell.")
    await pick(user, 1, 1, "Empty")
    await waitFor(async () =>
      expect((await loadPageReview(moved))?.cells[0].mark).toBe("empty")
    )
    expect(await photoText(await loadPageReview(moved))).toBe("moved")
    expect(await loadPageReview(key)).toBeUndefined()
  })

  it("works from memory with its photo when IndexedDB is missing, and nothing throws", async () => {
    vi.stubGlobal("indexedDB", undefined)
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderFlow(
      source({ frameFromFile: async () => photoOf("memory") }),
      cut,
      onClose
    )
    await user.upload(await screen.findByLabelText("Choose a photo"), rawFile())
    await user.click(
      await screen.findByRole("button", { name: "Cut and match" })
    )
    expect(
      await screen.findByText(/This review is not being saved/)
    ).toBeTruthy()

    await answerAll(fourConfident)
    await waitFor(() =>
      expect(confirmButton().hasAttribute("disabled")).toBe(false)
    )
    await user.click(confirmButton())
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })
})
