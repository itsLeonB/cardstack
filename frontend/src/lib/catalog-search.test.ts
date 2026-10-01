import { describe, expect, it } from "vitest"
import { catalogSearchSchema } from "./catalog-search"

describe("catalogSearchSchema", () => {
  it("wraps a legacy single value into an array", () => {
    expect(catalogSearchSchema.parse({ rarityId: "r1" }).rarityId).toEqual([
      "r1",
    ])
  })

  it("keeps multiple values and stringifies numeric-looking ones", () => {
    const parsed = catalogSearchSchema.parse({
      tag: [1, "V"],
      expansionSetId: ["a", "b"],
    })
    expect(parsed.tag).toEqual(["1", "V"])
    expect(parsed.expansionSetId).toEqual(["a", "b"])
  })

  it("leaves absent filters undefined", () => {
    expect(catalogSearchSchema.parse({}).category).toBeUndefined()
  })
})
