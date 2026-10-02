import { Link } from "@tanstack/react-router"
import type { LinkProps } from "@tanstack/react-router"

export type Crumb = { label: string; link?: LinkProps }

/**
 * Trail of ancestors ending in the current page: pass every crumb but the last
 * with a `link`; the last renders as the non-link current page. Plain links in
 * a `nav`, no `ol`/`li`, because e2e specs count `listitem` for results.
 */
export function Breadcrumbs({ crumbs }: { crumbs: Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb" className="flex flex-wrap items-center gap-x-2 text-sm">
      {crumbs.map(({ label, link }, index) => {
        const isCurrent = index === crumbs.length - 1
        return (
          <span key={label} className="flex min-w-0 items-center gap-x-2">
            {index > 0 && (
              <span aria-hidden="true" className="text-muted-foreground">
                /
              </span>
            )}
            {link && !isCurrent ? (
              <Link
                {...link}
                // Ancestors prefix-match the current URL; without this the router marks them aria-current too.
                activeOptions={{ exact: true }}
                className="rounded-4xl underline underline-offset-2 outline-none hover:no-underline focus-visible:ring-3 focus-visible:ring-ring/50"
              >
                {label}
              </Link>
            ) : (
              <span aria-current={isCurrent ? "page" : undefined} className="truncate font-medium">
                {label}
              </span>
            )}
          </span>
        )
      })}
    </nav>
  )
}
