import { describe, expect, it } from "vitest"
import { buildRobots, buildSitemap } from "./seo-files"

describe("seo files", () => {
  it("lists only the landing page and the Catalog index in the sitemap", () => {
    const locs = [...buildSitemap("https://cards.example").matchAll(/<loc>(.*?)<\/loc>/g)].map((m) => m[1])

    expect(locs).toEqual(["https://cards.example/", "https://cards.example/catalog"])
  })

  it("appends the sitemap URL to the robots rules", () => {
    const robots = buildRobots("User-agent: *\nDisallow: /account\n", "https://cards.example")

    expect(robots).toBe("User-agent: *\nDisallow: /account\n\nSitemap: https://cards.example/sitemap.xml\n")
  })
})
