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
import { ScanPageFlow } from "./page-scan"
import type { FrameSource } from "@/lib/frame-source"
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

/** Answers the oldest held match. */
function answer(reply: Reply) {
  held.shift()!(
    reply === "fail" ? json({ title: "boom" }, 503) : json({ data: reply })
  )
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

function renderFlow(
  frames = source(),
  cutPhoto: CutPhoto = cut,
  onClose = vi.fn(),
  current = page
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanPageFlow
        collectionId="col-1"
        page={current}
        source={frames}
        cut={cutPhoto}
        onClose={onClose}
      />
    </QueryClientProvider>
  )
}

/** Picks a photo and cuts it, leaving the review with all four cells matching. */
async function scanPhoto(user: ReturnType<typeof userEvent.setup>) {
  await user.upload(
    screen.getByLabelText("Choose a photo"),
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
      screen.getByLabelText("Choose a photo"),
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

  it("shows each cell's match: confident, unsure with the top candidate, no good candidate, failed", async () => {
    const user = userEvent.setup()
    renderFlow()
    await scanPhoto(user)
    await waitFor(() => expect(matchCalls).toBe(3))

    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      { confident: false, candidates: [candidate(2), candidate(3)] },
      { confident: false, candidates: [] },
      "fail",
    ])

    const unsure = await within(cell(1, 2)).findByRole("combobox")
    const chosen = () =>
      within(cell(1, 2)).getByRole("option", { selected: true }).textContent
    expect(chosen()).toBe("Card 2 #2")
    await user.selectOptions(unsure, "card-3")
    expect(chosen()).toBe("Card 3 #3")
    expect(
      await within(cell(2, 1)).findByText("No good candidate")
    ).toBeTruthy()
    expect(within(cell(2, 2)).getByRole("alert").textContent).toBe(
      "Could not match this cell."
    )
    expect(within(cell(1, 1)).getByText("Card 1")).toBeTruthy()
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

  it("saves each cell as the scan found it, with hand marks overriding it", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderFlow(source(), cut, onClose)
    await scanPhoto(user)
    await answerAll([
      { confident: true, candidates: [candidate(1)] },
      "fail",
      { confident: false, candidates: [] },
      { confident: true, candidates: [candidate(2)] },
    ])

    await user.click(within(cell(1, 1)).getByRole("button", { name: "Empty" }))
    const unidentified = within(cell(2, 2)).getByRole("button", {
      name: "Unidentified",
    })
    await user.click(unidentified)
    await user.click(unidentified)
    await user.click(confirmButton())

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(saves).toEqual([
      {
        slots: [
          { state: "empty" },
          { state: "unidentified" },
          { state: "unidentified" },
          { state: "card", cardId: "card-2" },
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

    expect(
      await within(cell(1, 2)).findByText("Changes this Slot")
    ).toBeTruthy()
    expect(within(cell(1, 1)).queryByText("Changes this Slot")).toBeNull()
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

  it("asks before leaving a review that has edits", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderFlow(source(), cut, onClose)
    await scanPhoto(user)
    await answerAll(fourConfident)
    await user.click(within(cell(1, 1)).getByRole("button", { name: "Empty" }))

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
})
