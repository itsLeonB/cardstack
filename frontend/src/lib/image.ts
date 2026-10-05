/**
 * Turns the API's hosted-image address into Cloudflare Image Transformations
 * addresses (ADR-0016). Only addresses on `VITE_IMAGE_HOST` are rewritten; an
 * empty address, an off-host one (local development) or an unset host comes
 * back unchanged, so nothing breaks where hosting isn't configured. Only cards
 * are rewritten: Expansion Set covers are pre-sized at host time and used as-is
 * (ADR-0017).
 */

export type ImageVariant = "cardTile" | "cardDetail"

// Widths per variant. A card tile gets a second, 2x source for high-density screens.
const WIDTHS = {
  cardTile: [240, 480],
  cardDetail: [720],
} satisfies Record<ImageVariant, number[]>

export type ImageSources = { src: string; srcSet?: string }

/** Origin of the image host; read per call so tests can stub it. */
function imageOrigin(): string | undefined {
  const host = import.meta.env.VITE_IMAGE_HOST
  if (!host) return undefined
  try {
    return new URL(host).origin
  } catch {
    return undefined
  }
}

export function imageSources(url: string, variant: ImageVariant): ImageSources {
  const origin = imageOrigin()
  if (!url || !origin) return { src: url }

  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    return { src: url }
  }
  if (parsed.origin !== origin) return { src: url }

  // `onerror=redirect` serves the original when the free transformation allowance is spent.
  // Only the path is kept: hosted keys never carry a query string.
  const addresses = WIDTHS[variant].map(
    (width) =>
      `${origin}/cdn-cgi/image/width=${width},format=auto,onerror=redirect${parsed.pathname}`
  )
  if (addresses.length === 1) return { src: addresses[0] }
  // One address per screen density, 1x upward.
  return {
    src: addresses[0],
    srcSet: addresses.map((address, i) => `${address} ${i + 1}x`).join(", "),
  }
}
