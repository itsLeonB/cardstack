import { useState } from "react"
import { useRouterState } from "@tanstack/react-router"
import { cn } from "cn"
import { ScanLayerContext } from "@/lib/scan-layer"
import { MobileNavigation, SiteHeader } from "./site-header"
import { SiteFooter } from "./site-footer"
import { SkipLink } from "./skip-link"

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  // A route that is itself a scan page, or one whose scan layer is open.
  const isScan = pathname.endsWith("/scan")
  const [scanLayer, setScanLayer] = useState(false)
  const hideMobileBar = isScan || scanLayer

  return (
    <ScanLayerContext.Provider value={setScanLayer}>
      <div
        className={cn(
          "flex min-h-svh flex-col",
          !hideMobileBar &&
            "pb-[calc(4rem+env(safe-area-inset-bottom,0px))] md:pb-0"
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
        {!hideMobileBar && <MobileNavigation />}
      </div>
    </ScanLayerContext.Provider>
  )
}
