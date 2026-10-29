import { readFileSync } from "node:fs"
import { describe, expect, it } from "vitest"

// The auth and shared pages' client-side noindex <meta> is invisible to crawlers that
// don't run scripts, and Vercel serves one static shell for every URL, so the
// header is the only server-visible directive.
describe("vercel.json", () => {
  it.each(["/auth/:path*", "/s/:path*"])(
    "sends X-Robots-Tag: noindex for %s",
    (source) => {
      // SAFETY: vercel.json is a checked-in file whose shape this very test asserts.
      const config = JSON.parse(readFileSync("vercel.json", "utf8")) as {
        headers: { source: string; headers: { key: string; value: string }[] }[]
      }

      const rule = config.headers.find((entry) => entry.source === source)

      expect(rule?.headers).toContainEqual({
        key: "X-Robots-Tag",
        value: "noindex",
      })
    }
  )
})
