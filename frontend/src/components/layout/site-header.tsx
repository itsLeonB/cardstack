import { useEffect, useState } from "react"
import type { ReactNode } from "react"
import { Link, useHydrated, useRouterState } from "@tanstack/react-router"
import {
  RiBookOpenLine,
  RiExchangeLine,
  RiHeartLine,
  RiLoginBoxLine,
  RiMoreLine,
  RiStackLine,
  RiUserAddLine,
  RiUserLine,
} from "@remixicon/react"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useSession } from "@/lib/session"
import { cn } from "cn"
import { useNavLinks } from "./nav-links"
import type { NavLink } from "./nav-links"
import { ThemeToggle } from "./theme-toggle"
import { UserMenu } from "./user-menu"
import { Wordmark } from "./wordmark"

export function SiteHeader() {
  const links = useNavLinks().filter((link) => link.to !== "/account")
  const { isAuthenticated } = useSession()
  const hydrated = useHydrated()

  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 pt-[env(safe-area-inset-top,0px)] backdrop-blur">
      {/* The SPA shell is prerendered at "/" and React keeps the attributes it
          hydrates, so the links' aria-current would stay that of "/". Remount
          once hydrated to render them for the real location. */}
      <div
        key={String(hydrated)}
        className="mx-auto flex h-14 max-w-5xl items-center gap-2 px-4"
      >
        <Wordmark />
        <nav
          aria-label="Main"
          className="ml-4 hidden items-center gap-1 md:flex"
        >
          {links.map((link) => (
            <NavLinkItem key={link.to} link={link} />
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1">
          <ThemeToggle />
          {isAuthenticated && <UserMenu />}
        </div>
      </div>
    </header>
  )
}

const mobileIcons = {
  "/catalog": <RiBookOpenLine aria-hidden="true" className="size-5" />,
  "/collections": <RiStackLine aria-hidden="true" className="size-5" />,
  "/wishlist": <RiHeartLine aria-hidden="true" className="size-5" />,
  "/trades": <RiExchangeLine aria-hidden="true" className="size-5" />,
  "/auth/login": <RiLoginBoxLine aria-hidden="true" className="size-5" />,
  "/auth/register": <RiUserAddLine aria-hidden="true" className="size-5" />,
  "/inventory": <RiStackLine aria-hidden="true" className="size-5" />,
  "/account": <RiUserLine aria-hidden="true" className="size-5" />,
} satisfies Record<NavLink["to"], ReactNode>

export function MobileNavigation() {
  const links = useNavLinks()
  const hydrated = useHydrated()
  const primaryLinks = links.filter((link) => !link.more)
  const moreLinks = links.filter((link) => link.more)
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const [open, setOpen] = useState(false)
  const [isDesktop, setIsDesktop] = useState(false)

  useEffect(() => {
    const media = window.matchMedia("(min-width: 768px)")
    const syncViewport = () => {
      setIsDesktop(media.matches)
      if (media.matches) setOpen(false)
    }
    syncViewport()
    media.addEventListener("change", syncViewport)
    return () => media.removeEventListener("change", syncViewport)
  }, [])

  const moreIsCurrent =
    pathname === "/account" || pathname.startsWith("/inventory")

  return (
    <nav
      aria-label="Bottom navigation"
      className="fixed inset-x-0 bottom-0 z-40 border-t bg-background/95 pr-[max(0.25rem,env(safe-area-inset-right,0px))] pb-[env(safe-area-inset-bottom,0px)] pl-[max(0.25rem,env(safe-area-inset-left,0px))] backdrop-blur md:hidden"
    >
      <div
        key={String(hydrated)}
        className="mx-auto grid max-w-5xl auto-cols-fr grid-flow-col"
      >
        {primaryLinks.map((link) => (
          <Link
            key={link.to}
            to={link.to}
            className="relative flex min-h-16 min-w-0 flex-col items-center justify-center gap-1 px-1 text-foreground outline-none after:absolute after:inset-x-1/3 after:bottom-1 after:h-0.5 after:rounded-full after:bg-foreground after:opacity-0 focus-visible:ring-3 focus-visible:ring-ring/50 [&[aria-current=page]]:after:opacity-100"
          >
            {mobileIcons[link.to]}
            <span className="text-center text-xs leading-tight break-words">
              {link.label}
            </span>
          </Link>
        ))}
        {moreLinks.length > 0 && (
          <DropdownMenu open={open && !isDesktop} onOpenChange={setOpen}>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  aria-label="More"
                  aria-current={moreIsCurrent ? "page" : undefined}
                  className="relative flex min-h-16 w-full flex-col items-center justify-center gap-1 rounded-none px-1 text-foreground after:absolute after:inset-x-1/3 after:bottom-1 after:h-0.5 after:rounded-full after:bg-foreground after:opacity-0 [&[aria-current=page]]:after:opacity-100"
                />
              }
            >
              <RiMoreLine aria-hidden="true" className="size-5" />
              <span className="text-xs leading-tight">More</span>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" side="top" className="mb-2">
              {moreLinks.map((link) => (
                <DropdownMenuItem key={link.to} render={<Link to={link.to} />}>
                  {link.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </nav>
  )
}

/**
 * One header link, shared by the desktop nav and the mobile menu. `onNavigate`
 * closes the mobile menu; `className` lets the mobile menu widen the primary
 * (button-styled) link to fit its column.
 */
function NavLinkItem({
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
