export const SITE_NAME = "Cardstack"

export const NOINDEX_META = { name: "robots", content: "noindex" }

/** Route `head` for a page: "<name> · Cardstack". */
export function pageHead(name: string) {
  return { meta: [{ title: `${name} · ${SITE_NAME}` }] }
}
