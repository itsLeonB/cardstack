import { useState } from "react"

/**
 * A Series section's `<h2>`: the hosted logo when there is one, else the name
 * as text. The name stays in the heading either way (visually hidden behind a
 * logo), so the heading, the section's `aria-labelledby` and the outline don't
 * change. The logo is the hosted file as-is, never through `imageSources()`:
 * it is pre-sized (80px tall at source, shown at 40px for 2x screens) and a
 * `/cdn-cgi/image/` rewrite would transform it again.
 */
export function SeriesHeading({
  id,
  name,
  imageUrl,
}: {
  id: string
  name: string
  imageUrl: string
}) {
  const [imageFailed, setImageFailed] = useState(false)

  if (!imageUrl || imageFailed) {
    return (
      <h2 id={id} className="font-heading text-lg font-medium">
        {name}
      </h2>
    )
  }

  return (
    <h2 id={id}>
      <span className="sr-only">{name}</span>
      {/* Decorative: the name is in the heading. The fixed height reserves the
          slot; the width follows the logo's aspect ratio. */}
      <img
        src={imageUrl}
        alt=""
        height={40}
        loading="lazy"
        className="h-10 w-auto max-w-full"
        onError={() => setImageFailed(true)}
      />
    </h2>
  )
}
