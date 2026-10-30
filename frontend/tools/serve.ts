// Serves the built client output the way Vercel serves this package: a real
// static file wins, otherwise the first matching rewrite, and every header rule
// whose source matches the path adds its headers. The rules come from this
// package's own vercel.json, so they cannot drift from what Vercel applies.
// Node APIs only, because vitest runs this on Node; Bun runs it as the entrypoint.
import { readFileSync } from "node:fs"
import { readFile, stat } from "node:fs/promises"
import { extname, join, resolve, sep } from "node:path"

export interface VercelConfig {
  headers?: { source: string; headers: { key: string; value: string }[] }[]
  rewrites?: { source: string; destination: string }[]
}

const CONTENT_TYPES = new Map([
  [".css", "text/css; charset=utf-8"],
  [".html", "text/html; charset=utf-8"],
  [".ico", "image/x-icon"],
  [".jpg", "image/jpeg"],
  [".js", "text/javascript; charset=utf-8"],
  [".json", "application/json"],
  [".png", "image/png"],
  [".svg", "image/svg+xml"],
  [".txt", "text/plain; charset=utf-8"],
  [".webmanifest", "application/manifest+json"],
  [".webp", "image/webp"],
  [".woff", "font/woff"],
  [".woff2", "font/woff2"],
  [".xml", "application/xml"],
])

// ponytail: covers the source forms these vercel.json files use (`/:name`, `/:name*`, `(.*)`).
// URLPattern would do it, but Node 22, which vitest runs on, does not ship it.
const TOKEN = /(\/:\w+\*|\/:\w+|\(\.\*\))/

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

function tokenPattern(token: string): string {
  if (token.startsWith("(")) return "(.*)"
  // `/:name*` matches the bare prefix too, so `/auth` matches `/auth/:path*`.
  return token.endsWith("*") ? "(?:/.*)?" : "/[^/]+"
}

export function sourceToRegExp(source: string): RegExp {
  // split with a capture group puts the tokens at odd indexes.
  const pattern = source
    .split(TOKEN)
    .map((part, index) => (index % 2 ? tokenPattern(part) : escapeRegExp(part)))
    .join("")
  return new RegExp(`^${pattern}$`)
}

async function findFile(
  root: string,
  path: string
): Promise<string | undefined> {
  const file = resolve(root, `.${path}`)
  if (!file.startsWith(root + sep)) return undefined
  const info = await stat(file).catch(() => undefined)
  return info?.isFile() ? file : undefined
}

export function createServeHandler(root: string, config: VercelConfig) {
  const base = resolve(root)
  const headerRules = (config.headers ?? []).map((rule) => ({
    pattern: sourceToRegExp(rule.source),
    headers: rule.headers,
  }))
  const rewriteRules = (config.rewrites ?? []).map((rule) => ({
    pattern: sourceToRegExp(rule.source),
    destination: rule.destination,
  }))

  return async (request: Request): Promise<Response> => {
    const { pathname } = new URL(request.url)
    const headers = new Headers()
    for (const rule of headerRules) {
      if (!rule.pattern.test(pathname)) continue
      for (const header of rule.headers)
        headers.append(header.key, header.value)
    }

    // ponytail: the rewrite destination is used literally, since both vercel.json files point at the shell.
    const rewrite = rewriteRules.find((rule) => rule.pattern.test(pathname))
    const file =
      (await findFile(base, pathname)) ??
      (await findFile(base, rewrite?.destination ?? pathname))
    if (!file) return new Response("Not found", { status: 404, headers })

    headers.set(
      "content-type",
      CONTENT_TYPES.get(extname(file)) ?? "application/octet-stream"
    )
    return new Response(await readFile(file), { headers })
  }
}

// Bun serves a default export of this shape when the file is the entrypoint.
// Importing it, as the tests do, starts nothing.
const dir = import.meta.dirname
// SAFETY: vercel.json is a checked-in file with the shape VercelConfig describes.
export default {
  hostname: "0.0.0.0",
  port: Number(process.env.PORT ?? 3000),
  fetch: createServeHandler(
    join(dir, "..", "dist", "client"),
    JSON.parse(
      readFileSync(join(dir, "..", "vercel.json"), "utf8")
    ) as VercelConfig
  ),
}
