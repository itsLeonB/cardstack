import { Link } from "@tanstack/react-router"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { NavLink } from "./nav-links"

export function NavLinkItem({
  link,
  onNavigate,
  className,
}: {
  link: NavLink
  onNavigate?: () => void
  className?: string
}) {
  return (
    <Link
      to={link.to}
      onClick={onNavigate}
      className={cn(
        link.primary
          ? buttonVariants({ size: "sm" })
          : "rounded-4xl px-4 py-2 text-sm font-medium text-muted-foreground outline-none hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 [&[aria-current=page]]:bg-muted [&[aria-current=page]]:text-foreground",
        className
      )}
    >
      {link.label}
    </Link>
  )
}
