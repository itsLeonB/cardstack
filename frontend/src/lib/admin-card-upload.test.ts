import { afterEach, describe, expect, it, vi } from "vitest"
import {
  MAX_IMAGE_BYTES,
  uploadCardImage,
  validateCardImage,
} from "./admin-card-upload"

afterEach(() => vi.unstubAllGlobals())

/** jsdom cannot decode images, so the browser's decoder reports these sizes. */
function stubDecoder(width = 720, height = 1005) {
  vi.stubGlobal(
    "createImageBitmap",
    vi.fn(async () => ({ width, height, close: () => {} }))
  )
}

function fileOfSize(size: number, type = "image/png") {
  const file = new File(["x"], "card", { type })
  Object.defineProperty(file, "size", { value: size })
  return file
}

interface Call {
  method: string
  url: string
  headers: Headers
  body: unknown
}

const SIGNED_URL = "https://bucket.test/cards/c1?X-Amz-Signature=abc"

function json<T>(status: number, body: T) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

/** A fetch fake: `put` answers the browser's PUT to the bucket. */
function stubFetch(
  put: () => Response = () => new Response(null, { status: 200 })
) {
  const calls: Call[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const call = {
        method: init?.method ?? "GET",
        url,
        headers: new Headers(init?.headers),
        body: init?.body,
      }
      calls.push(call)
      if (url === SIGNED_URL) return put()
      if (url.endsWith("/image-upload")) {
        return json(200, {
          data: {
            url: SIGNED_URL,
            contentType: "image/png",
            expiresAt: "2026-10-12T10:00:00Z",
          },
        })
      }
      return json(200, { data: { id: "c1", imageUrl: "https://img.test/c1" } })
    })
  )
  return calls
}

const png = () => new File(["x"], "card.png", { type: "image/png" })

describe("uploadCardImage", () => {
  it("asks for an address, PUTs the file to it, then confirms", async () => {
    const calls = stubFetch()
    stubDecoder()
    const file = png()

    const card = await uploadCardImage("c1", file)

    expect(
      calls.map((c) => [c.method, c.url.replace(/^.*\/admin/, "/admin")])
    ).toEqual([
      ["POST", "/admin/cards/c1/image-upload"],
      ["PUT", SIGNED_URL],
      ["POST", "/admin/cards/c1/image-confirm"],
    ])
    expect(JSON.parse(String(calls[0].body))).toEqual({
      contentType: "image/png",
    })
    expect(calls[1].body).toBe(file)
    expect(calls[1].headers.get("content-type")).toBe("image/png")
    expect(calls[1].headers.has("authorization")).toBe(false)
    expect(card.imageUrl).toBe("https://img.test/c1")
  })

  it("sends nothing for a file that is not a JPEG, PNG or WebP", async () => {
    const calls = stubFetch()

    await expect(
      uploadCardImage("c1", new File(["x"], "card.gif", { type: "image/gif" }))
    ).rejects.toThrow(/JPEG, PNG or WebP/)

    expect(calls).toEqual([])
  })

  it("sends nothing for a file the checks refuse", async () => {
    const calls = stubFetch()
    stubDecoder(1000, 500)

    await expect(uploadCardImage("c1", png())).rejects.toThrow(/portrait/)

    expect(calls).toEqual([])
  })

  it("does not confirm when the bucket refuses the PUT", async () => {
    stubDecoder()
    const calls = stubFetch(() => new Response(null, { status: 403 }))

    await expect(uploadCardImage("c1", png())).rejects.toThrow(/403/)

    expect(calls.map((c) => c.method)).toEqual(["POST", "PUT"])
  })

  it("surfaces the API's reason when the confirm step refuses the file", async () => {
    stubFetch()
    stubDecoder()
    const fetchMock = vi.mocked(fetch)
    const base = fetchMock.getMockImplementation()!
    fetchMock.mockImplementation(async (input, init) =>
      String(input).endsWith("/image-confirm")
        ? json(422, { status: 422, detail: "not an image" })
        : base(input, init)
    )

    await expect(uploadCardImage("c1", png())).rejects.toThrow("not an image")
  })
})

describe("validateCardImage", () => {
  it("accepts a card-shaped JPEG, PNG or WebP within the limits", async () => {
    stubDecoder()
    for (const type of ["image/jpeg", "image/png", "image/webp"]) {
      await expect(validateCardImage(fileOfSize(1000, type))).resolves.toBe(
        type
      )
    }
  })

  it("refuses another type, naming the allowed ones", async () => {
    stubDecoder()
    await expect(
      validateCardImage(fileOfSize(1000, "image/gif"))
    ).rejects.toThrow(/JPEG, PNG or WebP/)
  })

  it("refuses a file over the size limit and accepts one at it", async () => {
    stubDecoder()
    await expect(
      validateCardImage(fileOfSize(MAX_IMAGE_BYTES + 1))
    ).rejects.toThrow(/10 MB/)
    await expect(validateCardImage(fileOfSize(MAX_IMAGE_BYTES))).resolves.toBe(
      "image/png"
    )
  })

  it.each([
    ["landscape", 1005, 720, /portrait/],
    ["square", 800, 800, /portrait/],
    ["too small", 150, 210, /at least 300/],
    ["too large", 4500, 6280, /at most 6000/],
  ])("refuses a %s image", async (_name, width, height, reason) => {
    stubDecoder(width, height)
    await expect(validateCardImage(fileOfSize(1000))).rejects.toThrow(reason)
  })

  it("refuses a file the browser cannot decode", async () => {
    vi.stubGlobal(
      "createImageBitmap",
      vi.fn(async () => {
        throw new Error("bad")
      })
    )
    await expect(validateCardImage(fileOfSize(1000))).rejects.toThrow(
      /could not be read/
    )
  })
})
