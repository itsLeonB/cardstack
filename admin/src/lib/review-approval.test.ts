import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { approveCorrection } from "@/lib/review-approval"
import type { CardSummary, ReviewItem } from "@/generated/models"
import type { Pixels } from "@/lib/photo-pixels"

// Approval through the generated client, with the bucket's fetch and the
// browser's pixels stubbed. The calls are recorded in order, because the API
// refuses an approval whose object has not been uploaded yet.
const api = vi.hoisted(() => {
  const calls: string[] = []
  return {
    calls,
    getReviewPhoto: vi.fn(),
    requestReviewUpload: vi.fn(),
    approveReview: vi.fn(),
    readPixels: vi.fn(),
    encodePixelsJpeg: vi.fn(),
  }
})
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/admin/admin", () => ({
  getReviewPhoto: api.getReviewPhoto,
  requestReviewUpload: api.requestReviewUpload,
  approveReview: api.approveReview,
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/photo-pixels", () => ({
  readPixels: api.readPixels,
  encodePixelsJpeg: api.encodePixelsJpeg,
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

const scan: ReviewItem = {
  id: "scan",
  kind: "switched",
  source: "card_scan",
  createdAt: "2026-10-01T10:00:00Z",
  card: pikachu,
  scores: null,
}

const whole = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]

// The right-hand cell of a one-by-two Page, so its cut is 19 by 19 pixels.
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
  card: pikachu,
  scores: null,
  binder,
}

const width = 40
const height = 20
const halves = new Uint8ClampedArray(width * height * 4)
for (let at = 0; at < halves.length; at += 4) {
  const x = (at / 4) % width
  halves.set([x < width / 2 ? 0 : 255, 0, 0, 255], at)
}
const page: Pixels = { width, height, pixels: halves }

const upload = {
  status: 200,
  data: {
    data: {
      key: "reviewed/scan.jpg",
      url: "https://upload.test/scan",
      expiresAt: "2026-10-31T12:05:00Z",
    },
  },
}

// What the bucket received: the body as text, and the Content-Type it was signed for.
let put: { body: string; contentType: string | null } | null

// The stored photo is served for a GET; a PUT is recorded and answered with
// `putStatus`, the way the bucket answers.
function stubBucket(stored: string, putStatus = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, init?: RequestInit) => {
      if (init?.method === "PUT") {
        api.calls.push("put")
        put = {
          body: await new Response(init.body ?? null).text(),
          contentType: new Headers(init.headers).get("Content-Type"),
        }
        return new Response(null, { status: putStatus })
      }
      api.calls.push("read")
      return new Response(stored)
    })
  )
}

beforeEach(() => {
  api.calls.length = 0
  put = null
  api.getReviewPhoto.mockImplementation(async () => {
    api.calls.push("photo")
    return {
      status: 200,
      data: {
        data: {
          url: "https://photos.test/photo.jpg",
          expiresAt: "2026-10-31T12:05:00Z",
        },
      },
    }
  })
  api.requestReviewUpload.mockImplementation(async () => {
    api.calls.push("upload")
    return upload
  })
  api.approveReview.mockImplementation(async () => {
    api.calls.push("approve")
    return { status: 204, data: undefined }
  })
  api.readPixels.mockResolvedValue(page)
  api.encodePixelsJpeg.mockImplementation(() => new Response("cut-jpeg").blob())
})

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
})

describe("approving a Correction's photo", () => {
  it("uploads the card-scan photo as stored, then approves the key the upload named", async () => {
    stubBucket("scan-bytes")

    await approveCorrection(scan)

    expect(api.calls).toEqual(["photo", "read", "upload", "put", "approve"])
    expect(put).toEqual({ body: "scan-bytes", contentType: "image/jpeg" })
    expect(api.approveReview).toHaveBeenCalledWith("scan", {
      key: "reviewed/scan.jpg",
    })
  })

  it("uploads a binder cell as the JPEG of the cut the review shows, then approves", async () => {
    stubBucket("")

    await approveCorrection(cell)

    expect(api.readPixels).toHaveBeenCalledWith("https://photos.test/photo.jpg")
    expect(api.encodePixelsJpeg).toHaveBeenCalledWith(
      expect.objectContaining({ width: 19, height: 19 })
    )
    expect(api.calls).toEqual(["photo", "upload", "put", "approve"])
    expect(put).toEqual({ body: "cut-jpeg", contentType: "image/jpeg" })
  })

  it("refuses a binder cell with no corners before asking for an upload address", async () => {
    const cornerless: ReviewItem = {
      ...cell,
      binder: { ...binder, corners: null },
    }

    await expect(approveCorrection(cornerless)).rejects.toMatchObject({
      refresh: false,
    })
    expect(api.requestReviewUpload).not.toHaveBeenCalled()
    expect(api.approveReview).not.toHaveBeenCalled()
  })

  it("does not approve when the upload address is refused, and asks for the queue again", async () => {
    api.requestReviewUpload.mockResolvedValue({
      status: 404,
      data: { detail: "correction not found" },
    })
    stubBucket("scan-bytes")

    await expect(approveCorrection(scan)).rejects.toMatchObject({
      message: "This photo is no longer waiting for review.",
      refresh: true,
    })
    expect(api.calls).not.toContain("put")
    expect(api.approveReview).not.toHaveBeenCalled()
  })

  it("does not approve when the bucket refuses the photo", async () => {
    stubBucket("scan-bytes", 403)

    await expect(approveCorrection(scan)).rejects.toThrow(
      "The image bucket refused the upload (403)."
    )
    expect(api.approveReview).not.toHaveBeenCalled()
  })

  it("tells the Admin another Admin decided the photo, and asks for the queue again", async () => {
    api.approveReview.mockResolvedValue({
      status: 409,
      data: { detail: "this Correction is already reviewed" },
    })
    stubBucket("scan-bytes")

    await expect(approveCorrection(scan)).rejects.toMatchObject({
      message:
        "Cannot approve: another Admin decided this photo, or its chosen Card was deleted.",
      refresh: true,
    })
  })

  it.each([
    [422, "the uploaded photo is not a JPEG"],
    [500, "the embedding service failed"],
    [503, "photo storage is not configured"],
  ])(
    "keeps the photo in the queue and shows the reason when approval answers %i",
    async (status, detail) => {
      api.approveReview.mockResolvedValue({ status, data: { detail } })
      stubBucket("scan-bytes")

      await expect(approveCorrection(scan)).rejects.toMatchObject({
        message: `Could not approve the photo: ${detail}. Try again.`,
        refresh: false,
      })
    }
  )
})
