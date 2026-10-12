import { afterEach, describe, expect, it, vi } from "vitest"
import { uploadCardImage } from "./admin-card-upload"

afterEach(() => vi.unstubAllGlobals())

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

  it("does not confirm when the bucket refuses the PUT", async () => {
    const calls = stubFetch(() => new Response(null, { status: 403 }))

    await expect(uploadCardImage("c1", png())).rejects.toThrow(/403/)

    expect(calls.map((c) => c.method)).toEqual(["POST", "PUT"])
  })

  it("surfaces the API's reason when the confirm step refuses the file", async () => {
    stubFetch()
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
