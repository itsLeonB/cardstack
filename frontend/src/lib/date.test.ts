import { describe, expect, it } from "vitest"
import { formatAge, formatReleaseDate } from "./date"

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

describe("formatAge", () => {
  it("says just now under a minute", () => {
    expect(formatAge(new Date().toISOString())).toBe("just now")
  })

  it("counts minutes, hours and days", () => {
    const now = Date.now()
    expect(formatAge(new Date(now - 5 * 60_000).toISOString())).toBe(
      "5 minutes ago"
    )
    expect(formatAge(new Date(now - 3 * 3_600_000).toISOString())).toBe(
      "3 hours ago"
    )
    expect(formatAge(new Date(now - 2 * 86_400_000).toISOString())).toBe(
      "2 days ago"
    )
  })

  it("uses singular units", () => {
    const now = Date.now()
    expect(formatAge(new Date(now - 61_000).toISOString())).toBe("1 minute ago")
    expect(formatAge(new Date(now - 3_600_000).toISOString())).toBe(
      "1 hour ago"
    )
    expect(formatAge(new Date(now - 86_400_000).toISOString())).toBe(
      "1 day ago"
    )
  })

  it("shows the calendar date past a month", () => {
    expect(formatAge("2011-03-03T00:00:00Z")).toBe("March 3, 2011")
  })

  it("returns null when the timestamp is absent, unparseable or future", () => {
    expect(formatAge(undefined)).toBeNull()
    expect(formatAge("not-a-date")).toBeNull()
    expect(formatAge(new Date(Date.now() + 60_000).toISOString())).toBeNull()
    expect(formatAge(new Date(Date.now() + 3_600_000).toISOString())).toBeNull()
  })
})
