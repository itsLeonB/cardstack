import {
  defaultParseSearch,
  defaultStringifySearch,
} from "@tanstack/react-router"
import { describe, expect, it } from "vitest"
import {
  catalogFilterSchema,
  catalogSearchSchema,
  guestFacets,
} from "./catalog-search"
import type { SeriesBrowseResult } from "@/generated/models"

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

  it("round-trips numeric-looking strings like 007 through the URL", () => {
    const url = defaultStringifySearch({ tag: ["007", "V"] })
    expect(catalogSearchSchema.parse(defaultParseSearch(url)).tag).toEqual([
      "007",
      "V",
    ])
  })
})

describe("catalogFilterSchema", () => {
  it("strips a legacy page param so an old link opens the first page", () => {
    const parsed = catalogFilterSchema.parse({ page: 3, name: "Pika" })
    expect(parsed).toEqual({ name: "Pika" })
  })
})

describe("guestFacets", () => {
  const set = (id: string) => ({
    id,
    code: id.toUpperCase(),
    name: `Set ${id}`,
    imageUrl: "",
    releaseDate: "2024-01-01",
  })
  const browse: SeriesBrowseResult = {
    series: [
      {
        id: "sr1",
        code: "SV",
        name: "Scarlet",
        imageUrl: "",
        expansionSets: [set("a")],
      },
      {
        id: "sr2",
        code: "SW",
        name: "Sword",
        imageUrl: "",
        expansionSets: null,
      },
    ],
    ungroupedExpansionSets: [set("b")],
  }

  it("lists every Expansion Set, under its Series, as the only options a Guest has", () => {
    expect(guestFacets(browse)).toEqual({
      expansionSets: [
        { ...set("a"), seriesId: "sr1", available: true },
        { ...set("b"), available: true },
      ],
      rarities: [],
      categories: [],
      tags: [],
    })
  })

  it("is undefined until the Series list has loaded", () => {
    expect(guestFacets(undefined)).toBeUndefined()
  })

  it("copes with a catalog that has no Series or sets", () => {
    expect(
      guestFacets({ series: null, ungroupedExpansionSets: null })?.expansionSets
    ).toEqual([])
  })
})
