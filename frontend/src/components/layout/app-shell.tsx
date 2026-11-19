import { useRouterState } from "@tanstack/react-router"
import { cn } from "cn"
import { MobileNavigation, SiteHeader } from "./site-header"
import { SiteFooter } from "./site-footer"
import { SkipLink } from "./skip-link"

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const isScan = pathname.endsWith("/scan")

  return (
    <div
      className={cn(
        "flex min-h-svh flex-col",
        !isScan && "pb-[calc(4rem+env(safe-area-inset-bottom,0px))] md:pb-0"
      )}
    >
      <SkipLink />
      <SiteHeader />
      <main
        id="main-content"
        tabIndex={-1}
        className={cn(
          "flex-1 outline-none",
          !isScan &&
            "[&_h1]:motion-safe:animate-in [&_h1]:motion-safe:duration-200 [&_h1]:motion-safe:slide-in-from-bottom-1"
        )}
      >
        {children}
      </main>
      <SiteFooter />
      {!isScan && <MobileNavigation />}
    </div>
  )
}
