import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { imageSources } from "./image"

const HOST = "https://img.example.com"
const original = `${HOST}/cards/abc-123`

beforeEach(() => vi.stubEnv("VITE_IMAGE_HOST", HOST))
afterEach(() => vi.unstubAllEnvs())

const transformed = (width: number) =>
  `${HOST}/cdn-cgi/image/width=${width},format=auto,onerror=redirect/cards/abc-123`

describe("imageSources", () => {
  it("sizes a card tile at 240 with a 480 source for 2x screens", () => {
    expect(imageSources(original, "cardTile")).toEqual({
      src: transformed(240),
      srcSet: `${transformed(240)} 1x, ${transformed(480)} 2x`,
    })
  })

  it("sizes the detail view at 720 with no source set", () => {
    expect(imageSources(original, "cardDetail")).toEqual({
      src: transformed(720),
    })
  })

  it("sizes an Expansion Set cover at 128 with no source set", () => {
    expect(imageSources(`${HOST}/expansion-sets/set-1`, "setCover")).toEqual({
      src: `${HOST}/cdn-cgi/image/width=128,format=auto,onerror=redirect/expansion-sets/set-1`,
    })
  })

  it("returns an empty address unchanged", () => {
    expect(imageSources("", "cardTile")).toEqual({ src: "" })
  })

  it("returns an address on another host unchanged", () => {
    const offHost = "https://other.example.net/cards/abc-123"

    expect(imageSources(offHost, "cardTile")).toEqual({ src: offHost })
  })

  it("returns the address unchanged when no image host is configured", () => {
    vi.stubEnv("VITE_IMAGE_HOST", "")

    expect(imageSources(original, "cardTile")).toEqual({ src: original })
  })

  it("returns an unparseable address unchanged", () => {
    expect(imageSources("not a url", "cardTile")).toEqual({ src: "not a url" })
  })

  it("tolerates a trailing slash on the configured host", () => {
    vi.stubEnv("VITE_IMAGE_HOST", `${HOST}/`)

    expect(imageSources(original, "cardDetail").src).toBe(transformed(720))
  })
})
