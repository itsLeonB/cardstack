import { describe, expect, it } from "vitest"
import { formatReleaseDate } from "./catalog"

describe("formatReleaseDate", () => {
  it("formats an RFC3339 date for display", () => {
    expect(formatReleaseDate("2011-03-03T00:00:00Z")).toBe("March 3, 2011")
  })

  it("returns null when the date is absent", () => {
    expect(formatReleaseDate(undefined)).toBeNull()
  })

  it("returns null when the date can't be parsed", () => {
    expect(formatReleaseDate("not-a-date")).toBeNull()
  })
})
