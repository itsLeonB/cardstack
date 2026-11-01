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
import { ApprovalRefused } from "@/lib/review-approval"
import type * as ReviewApproval from "@/lib/review-approval"

// The queue screen with the generated client mocked (the hooks answer from
// `api`). The cut itself is tested in `lib/binder-cell.test.ts`; here the
// browser's pixel reads are stubbed, so a binder cell's cut is checked by
// what gets painted.
const api = vi.hoisted(() => ({
  useQueue: vi.fn(),
  usePhoto: vi.fn(),
  photoRefetch: vi.fn(),
  reject: vi.fn(),
  approve: vi.fn(),
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
// Only the approval call is stubbed; the refusal class stays real, since the
// queue tells a refusal from a network failure by it.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/review-approval", async (importOriginal) => ({
  ...(await importOriginal<typeof ReviewApproval>()),
  approveCorrection: api.approve,
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
      data: { data: items, meta: { total, page: pageNumber, limit: 8 } },
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

function scanPhoto() {
  return within(itemNamed("Switched to another candidate")).getByRole("img", {
    name: "Photo from the scan",
  })
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

    fireEvent.error(scanPhoto())

    expect(api.photoRefetch).toHaveBeenCalledTimes(1)
  })

  it("offers Retry, not another fresh address, when the fresh one fails too", () => {
    renderQueue()

    fireEvent.error(scanPhoto())
    fireEvent.error(scanPhoto())

    expect(api.photoRefetch).toHaveBeenCalledTimes(1)
    const item = itemNamed("Switched to another candidate")
    within(item).getByText("The photo could not be loaded.")

    fireEvent.click(within(item).getByRole("button", { name: "Retry" }))
    fireEvent.error(scanPhoto())

    expect(api.photoRefetch).toHaveBeenCalledTimes(3)
  })

  it("asks for a fresh address again once a photo has loaded", () => {
    renderQueue()

    fireEvent.error(scanPhoto())
    fireEvent.load(scanPhoto())
    fireEvent.error(scanPhoto())

    expect(api.photoRefetch).toHaveBeenCalledTimes(2)
  })

  it("pages through the queue, 8 at a time", () => {
    api.useQueue.mockReturnValue(queue([scan, cell], 10))
    renderQueue()

    expect(api.useQueue).toHaveBeenLastCalledWith({ page: 1, limit: 8 })
    screen.getByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Next" }))

    expect(api.useQueue).toHaveBeenLastCalledWith({ page: 2, limit: 8 })
  })

  it("goes back to the last page when the queue shrinks under the one shown", () => {
    // The last photo of page 2 was decided: page 2 now answers empty, and the
    // queue fits on one page.
    api.useQueue.mockImplementation((params?: ListReviewQueueParams) =>
      params?.page === 2 ? queue([], 8, 2) : queue([scan, cell], 10)
    )
    renderQueue()

    fireEvent.click(screen.getByRole("button", { name: "Next" }))

    expect(api.useQueue).toHaveBeenLastCalledWith({ page: 1, limit: 8 })
    screen.getByText("Page 1 of 2")
    expect(screen.queryByText("Nothing is waiting for review.")).toBeNull()
  })

  it("approves a photo, then asks for the queue again so the photo leaves it", async () => {
    api.approve.mockResolvedValue(undefined)
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Approve",
      })
    )

    await vi.waitFor(() => expect(api.approve).toHaveBeenCalledWith(scan))
    await vi.waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ["review-queue"] })
    )
  })

  it("keeps the photo in the queue and says why approval was refused", async () => {
    api.approve.mockRejectedValue(
      new ApprovalRefused(
        "Could not approve the photo: the uploaded photo is not a JPEG. Try again.",
        false
      )
    )
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Approve",
      })
    )

    const alert = await within(
      itemNamed("Switched to another candidate")
    ).findByRole("alert")
    expect(alert.textContent).toBe(
      "Could not approve the photo: the uploaded photo is not a JPEG. Try again."
    )
    expect(invalidate).not.toHaveBeenCalled()
  })

  it("shows the conflict and asks for the queue again when another Admin decided the photo", async () => {
    const conflict = "Another Admin already decided this photo."
    api.approve.mockRejectedValue(new ApprovalRefused(conflict, true))
    const { invalidate } = renderQueue()

    fireEvent.click(
      within(itemNamed("Switched to another candidate")).getByRole("button", {
        name: "Approve",
      })
    )

    const alert = await within(
      itemNamed("Switched to another candidate")
    ).findByRole("alert")
    expect(alert.textContent).toBe(conflict)
    await vi.waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ["review-queue"] })
    )
  })

  it("holds both decisions on the photo while an approval is in flight", async () => {
    api.approve.mockReturnValue(new Promise(() => {}))
    renderQueue()
    const item = itemNamed("Switched to another candidate")

    fireEvent.click(within(item).getByRole("button", { name: "Approve" }))

    await within(item).findByText("Approving the photo…")
    expect(
      within(item)
        .getByRole("button", { name: "Approve" })
        .hasAttribute("disabled")
    ).toBe(true)
    expect(
      within(item)
        .getByRole("button", { name: "Reject" })
        .hasAttribute("disabled")
    ).toBe(true)
  })
})
