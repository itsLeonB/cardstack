export const SITE_NAME = "Cardstack"

// Baked in at build time (Vite replaces VITE_*). Link-preview images and the
// sitemap need an absolute URL, so both are skipped when it is unset.
export const SITE_URL = import.meta.env.VITE_SITE_URL?.replace(/\/$/, "")

export const NOINDEX_META = { name: "robots", content: "noindex" }

/** Route `head` for a page: "<name> · Cardstack", plus an optional meta description. */
export function pageHead(name: string, description?: string) {
  return {
    meta: [
      { title: `${name} · ${SITE_NAME}` },
      ...(description ? [{ name: "description", content: description }] : []),
    ],
  }
}
