import { useEffect, useRef, useState } from "react"
import { Link } from "@tanstack/react-router"
import { RiCloseLine, RiMenuLine } from "@remixicon/react"
import { Button, buttonVariants } from "@/components/ui/button"
import { useSession } from "@/lib/session"
import { cn } from "cn"
import { useNavLinks } from "./nav-links"
import type { NavLink } from "./nav-links"
import { ThemeToggle } from "./theme-toggle"
import { UserMenu } from "./user-menu"
import { Wordmark } from "./wordmark"

export function SiteHeader() {
  const links = useNavLinks()
  const { isAuthenticated } = useSession()
  const [open, setOpen] = useState(false)
  const toggleRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    const media = window.matchMedia("(min-width: 768px)")
    const onChange = () => {
      if (media.matches) setOpen(false)
    }
    media.addEventListener("change", onChange)
    return () => media.removeEventListener("change", onChange)
  }, [])

  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false)
        toggleRef.current?.focus()
      }
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [open])

  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-2 px-4">
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
          <Button
            ref={toggleRef}
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label="Menu"
            aria-expanded={open}
            aria-controls="mobile-nav"
            onClick={() => setOpen((value) => !value)}
          >
            {open ? <RiCloseLine /> : <RiMenuLine />}
          </Button>
        </div>
      </div>
      {open && (
        <nav
          id="mobile-nav"
          aria-label="Mobile"
          className="flex flex-col gap-1 border-t px-4 py-3 md:hidden"
        >
          {links.map((link) => (
            <NavLinkItem
              key={link.to}
              link={link}
              onNavigate={() => setOpen(false)}
              className={link.primary ? "w-fit" : undefined}
            />
          ))}
        </nav>
      )}
    </header>
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
