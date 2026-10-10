import { describe, expect, it } from "vitest"
import { fitLongSide, guideCrop } from "./frame-crop"

describe("guideCrop", () => {
  it("crops a landscape frame to the centred card guide", () => {
    expect(guideCrop(1280, 720)).toEqual({ x: 421, y: 54, w: 438, h: 612 })
  })

  it("crops a portrait frame to the centred card guide", () => {
    expect(guideCrop(720, 1280)).toEqual({ x: 68, y: 232, w: 584, h: 816 })
  })
})

describe("fitLongSide", () => {
  it("shrinks so the long side is the limit", () => {
    expect(fitLongSide(2000, 1000, 1024)).toEqual({ w: 1024, h: 512 })
    expect(fitLongSide(1000, 2000, 1024)).toEqual({ w: 512, h: 1024 })
  })

  it("never enlarges", () => {
    expect(fitLongSide(438, 612, 1024)).toEqual({ w: 438, h: 612 })
  })
})
