import { describe, expect, it } from "vitest"
import { pageHead } from "./site"

describe("pageHead", () => {
  it("prefixes the page name onto the site name", () => {
    expect(pageHead("Catalog").meta).toEqual([{ title: "Catalog · Cardstack" }])
  })

  it("adds a description only when given", () => {
    expect(pageHead("Catalog", "Browse sets.").meta).toContainEqual({
      name: "description",
      content: "Browse sets.",
    })
  })
})
