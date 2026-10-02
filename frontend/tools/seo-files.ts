// Build step: writes sitemap.xml and the robots.txt Sitemap line into the
// client build, since both need the absolute site origin (VITE_SITE_URL).
import { readFileSync, writeFileSync } from "node:fs"

// The sitemap assumes the Catalog index stays public. If guest catalog locking
// (ticket 24) changes that, drop "/catalog" here and disallow it in public/robots.txt,
// which holds the matching list of private paths.
const PUBLIC_PATHS = ["/", "/catalog"]

export function buildSitemap(origin: string) {
  const urls = PUBLIC_PATHS.map((path) => `  <url><loc>${origin}${path}</loc></url>`)
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls.join("\n")}\n</urlset>\n`
}

export function buildRobots(rules: string, origin: string) {
  return `${rules.trimEnd()}\n\nSitemap: ${origin}/sitemap.xml\n`
}

if (process.argv[1]?.endsWith("seo-files.ts")) {
  const origin = process.env.VITE_SITE_URL?.replace(/\/$/, "")
  if (!origin && process.env.VERCEL_ENV === "production") {
    // A production build without it would silently ship no preview image or sitemap.
    console.error("[seo-files] VITE_SITE_URL must be set for production builds")
    process.exit(1)
  } else if (!origin) {
    console.warn("[seo-files] VITE_SITE_URL is unset: skipping sitemap.xml and the robots.txt Sitemap line")
  } else {
    const out = "dist/client"
    writeFileSync(`${out}/sitemap.xml`, buildSitemap(origin))
    writeFileSync(`${out}/robots.txt`, buildRobots(readFileSync("public/robots.txt", "utf8"), origin))
    console.log(`[seo-files] wrote sitemap.xml and robots.txt for ${origin}`)
  }
}
