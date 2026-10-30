import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { afterAll, beforeAll, describe, expect, it } from "vitest"
import { createServeHandler } from "./serve"
import type { VercelConfig } from "./serve"

// The real vercel.json, so the header and rewrite rules tested here are the
// ones Vercel applies.
// SAFETY: vercel.json is a checked-in file with the shape VercelConfig describes.
const config = JSON.parse(
  readFileSync(join(import.meta.dirname, "..", "vercel.json"), "utf8")
) as VercelConfig

const SHELL = "<!doctype html><title>shell</title>"
const FAVICON = "<svg></svg>"
const STYLE = "body{margin:0}"

let root: string
let get: (path: string) => Promise<Response>

beforeAll(() => {
  root = mkdtempSync(join(tmpdir(), "serve-test-"))
  mkdirSync(join(root, "assets"))
  writeFileSync(join(root, "_shell.html"), SHELL)
  writeFileSync(join(root, "favicon.svg"), FAVICON)
  writeFileSync(join(root, "assets", "app.css"), STYLE)
  const handler = createServeHandler(root, config)
  get = (path) => handler(new Request(`http://localhost${path}`))
})

afterAll(() => {
  rmSync(root, { recursive: true, force: true })
})

describe("serve (user app)", () => {
  it("falls back to the shell page for an unknown path", async () => {
    const response = await get("/catalog/some-card")

    expect(response.status).toBe(200)
    expect(response.headers.get("content-type")).toContain("text/html")
    expect(await response.text()).toBe(SHELL)
  })

  it.each(["/auth", "/auth/x", "/auth/sign-in/deep", "/s/abc"])(
    "sends X-Robots-Tag: noindex on %s",
    async (path) => {
      const response = await get(path)

      expect(response.headers.get("x-robots-tag")).toBe("noindex")
    }
  )

  it.each(["/", "/catalog", "/collections", "/authz", "/s-other"])(
    "sends no X-Robots-Tag on %s",
    async (path) => {
      const response = await get(path)

      expect(response.headers.get("x-robots-tag")).toBeNull()
    }
  )

  it("serves a real static file over the fallback, with its headers", async () => {
    const favicon = await get("/favicon.svg")
    const style = await get("/assets/app.css")

    expect(await favicon.text()).toBe(FAVICON)
    expect(favicon.headers.get("content-type")).toContain("image/svg+xml")
    expect(await style.text()).toBe(STYLE)
    expect(style.headers.get("content-type")).toContain("text/css")
  })
})
