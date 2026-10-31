import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReviewQueue } from "@/components/admin/review-queue"
import type {
  CardSummary,
  ListReviewQueueParams,
  ReviewItem,
} from "@/generated/models"
import { paintPixels, readPixels } from "@/lib/photo-pixels"
import type { Pixels } from "@/lib/photo-pixels"

// The queue screen with the generated client mocked (the hooks answer from
// `api`). The cut itself is tested in `lib/binder-cell.test.ts`; here the
// browser's pixel reads are stubbed, so a binder cell's cut is checked by
// what gets painted.
const api = vi.hoisted(() => ({
  useQueue: vi.fn(),
  usePhoto: vi.fn(),
  photoRefetch: vi.fn(),
  reject: vi.fn(),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getListReviewQueueQueryKey: () => ["review-queue"],
  useListReviewQueue: (params?: ListReviewQueueParams) => api.useQueue(params),
  useGetReviewPhoto: (id: string) => api.usePhoto(id),
  useRejectReview: () => ({ mutate: api.reject, isPending: false }),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/photo-pixels", () => ({
  readPixels: vi.fn(),
  paintPixels: vi.fn(),
}))

const pikachu: CardSummary = {
  id: "pikachu",
  name: "Pikachu",
  localId: "25",
  category: "Pokemon",
  tags: [],
  illustrator: "",
  imageUrl: "https://cards.test/pikachu.webp",
  expansionSet: { id: "base", code: "BS", name: "Base", imageUrl: "" },
  rarity: { id: "common", code: "C", name: "Common" },
}

const raichu: CardSummary = {
  ...pikachu,
  id: "raichu",
  name: "Raichu",
  localId: "26",
  imageUrl: "https://cards.test/raichu.webp",
}

// A page photographed square-on, left half black and right half white: the
// user app's cut fixture (frontend/src/lib/page-cut.test.ts).
const width = 40
const height = 20
const halves = new Uint8ClampedArray(width * height * 4)
for (let at = 0; at < halves.length; at += 4) {
  const x = (at / 4) % width
  halves.set([x < width / 2 ? 0 : 255, 0, 0, 255], at)
}
const page: Pixels = { width, height, pixels: halves }

const whole = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]

const scan: ReviewItem = {
  id: "scan",
  kind: "switched",
  source: "card_scan",
  createdAt: "2026-10-01T10:00:00Z",
  card: pikachu,
  scores: [
    { cardId: "pikachu", score: 0.91 },
    { cardId: "raichu", score: 0.88 },
  ],
}

const binder = {
  page: 2,
  slotRow: 1,
  slotColumn: 2,
  gridRows: 1,
  gridColumns: 2,
  corners: whole,
}

const cell: ReviewItem = {
  id: "cell",
  kind: "reprint",
  source: "binder_page",
  createdAt: "2026-10-02T10:00:00Z",
  card: raichu,
  scores: null,
  binder,
}

function queue(items: ReviewItem[], total = items.length, pageNumber = 1) {
  return {
    data: {
      status: 200,
      data: { data: items, meta: { total, page: pageNumber, limit: 24 } },
    },
    isPending: false,
    isError: false,
  }
}

function renderQueue() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const invalidate = vi.spyOn(queryClient, "invalidateQueries")
  render(
    <QueryClientProvider client={queryClient}>
      <ReviewQueue />
    </QueryClientProvider>
  )
  return { invalidate }
}

function itemNamed(kindLabel: string) {
  return screen.getByRole("region", { name: kindLabel })
}

beforeEach(() => {
  api.useQueue.mockReturnValue(queue([scan, cell]))
  api.usePhoto.mockImplementation((id: string) => ({
    data: {
      status: 200,
      data: {
        data: {
          url: `https://photos.test/${id}.jpg`,
          expiresAt: "2026-10-31T12:05:00Z",
        },
      },
    },
    refetch: api.photoRefetch,
    isPending: false,
  }))
  vi.mocked(readPixels).mockResolvedValue(page)
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe("the review queue", () => {
  it("lists the waiting Corrections in the order the API gives, with the kind, the chosen Card and the scores", () => {
    renderQueue()

    const headings = screen
      .getAllByRole("heading", { level: 2 })
      .map((h) => h.textContent)
    expect(headings).toEqual([
      "Switched to another candidate",
      "Print from Other prints",
    ])
    const first = itemNamed("Switched to another candidate")
    within(first).getByText("Pikachu")
    within(first).getByText("0.910")
    within(first).getByText("raichu")
  })

  it("shows a card-scan photo as stored, from its presigned address", () => {
    renderQueue()

    const photo = within(itemNamed("Switched to another candidate")).getByRole(
      "img",
      { name: "Photo from the scan" }
    )
    expect(photo.getAttribute("src")).toBe("https://photos.test/scan.jpg")
  })

  it("shows a binder cell as the cut the matcher would embed, painted onto the canvas", async () => {
    renderQueue()

    await vi.waitFor(() => expect(paintPixels).toHaveBeenCalledTimes(1))
    expect(readPixels).toHaveBeenCalledWith("https://photos.test/cell.jpg")
    expect(paintPixels).toHaveBeenCalledWith(
      expect.any(HTMLCanvasElement),
      expect.objectContaining({ width: 19, height: 19 })
    )
  })

  it("says why a binder cell cannot be cut when it has no corners, and still offers Reject", async () => {
    const cornerless: ReviewItem = {
      ...cell,
      binder: { ...binder, corners: null },
    }
    api.useQueue.mockReturnValue(queue([cornerless]))
    renderQueue()

    await screen.findByText("This cell has no corners to cut.")
    expect(paintPixels).not.toHaveBeenCalled()
    const reject = within(itemNamed("Print from Other prints")).getByRole(
      "button",
      { name: "Reject" }
    )
    expect(reject.hasAttribute("disabled")).toBe(false)
  })

  it("removes a rejected photo by refreshing the queue", () => {
    api.reject.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 204 })
    )
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Reject",
      })
    )

    expect(api.reject).toHaveBeenCalledWith({ id: "scan" }, expect.any(Object))
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["review-queue"] })
  })

  it("tells the Admin when another Admin already decided the photo, and refreshes the queue", () => {
    api.reject.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 409 })
    )
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Reject",
      })
    )

    expect(
      within(itemNamed("Switched to another candidate")).getByRole("alert")
        .textContent
    ).toBe("Another Admin already decided this photo.")
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["review-queue"] })
  })

  it("says the photo is no longer waiting when the API answers not found", () => {
    api.reject.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 404 })
    )
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Reject",
      })
    )

    expect(
      within(itemNamed("Switched to another candidate")).getByRole("alert")
        .textContent
    ).toBe("This photo is no longer waiting for review.")
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["review-queue"] })
  })

  it("keeps the photo and says so when the rejection fails", () => {
    api.reject.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 500 })
    )
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Reject",
      })
    )

    expect(
      within(itemNamed("Switched to another candidate")).getByRole("alert")
        .textContent
    ).toBe("Could not reject the photo. Try again.")
    expect(invalidate).not.toHaveBeenCalled()
  })

  it("asks for a fresh photo address when the stored one has expired", () => {
    renderQueue()

    fireEvent.error(
      within(itemNamed("Switched to another candidate")).getByRole("img", {
        name: "Photo from the scan",
      })
    )

    expect(api.photoRefetch).toHaveBeenCalledTimes(1)
  })

  it("pages through the queue, 24 at a time", () => {
    api.useQueue.mockReturnValue(queue([scan, cell], 30))
    renderQueue()

    expect(api.useQueue).toHaveBeenLastCalledWith({ page: 1, limit: 24 })
    screen.getByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Next" }))

    expect(api.useQueue).toHaveBeenLastCalledWith({ page: 2, limit: 24 })
  })
})
