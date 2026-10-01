import { Link } from "@tanstack/react-router"

export function Wordmark() {
  return (
    <Link
      to="/"
      className="inline-flex items-center gap-2 rounded-md font-heading text-lg font-semibold outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      <svg viewBox="0 0 32 32" className="size-7" aria-hidden="true">
        <rect x="9" y="3" width="17" height="22" rx="3" className="fill-amber-700" transform="rotate(10 17 14)" />
        <rect x="6" y="5" width="17" height="22" rx="3" className="fill-amber-600" transform="rotate(-6 14 16)" />
        <rect x="7" y="7" width="17" height="22" rx="3" className="fill-amber-400" />
      </svg>
      Cardstack
    </Link>
  )
}
