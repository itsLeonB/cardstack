import { useRouterState } from "@tanstack/react-router"
import { useEffect, useState } from "react"

export function NavigationLoading() {
  const isNavigating = useRouterState({
    select: (state) =>
      state.isLoading &&
      state.resolvedLocation !== undefined &&
      state.location.href !== state.resolvedLocation.href,
  })
  const [showLoading, setShowLoading] = useState(false)

  useEffect(() => {
    if (!isNavigating) {
      setShowLoading(false)
      return
    }
    const timeout = window.setTimeout(() => setShowLoading(true), 200)
    return () => window.clearTimeout(timeout)
  }, [isNavigating])

  return (
    <div
      aria-live="polite"
      aria-atomic="true"
      className="pointer-events-none fixed top-[calc(env(safe-area-inset-top,0px)+0.5rem)] left-1/2 z-50 -translate-x-1/2"
    >
      {isNavigating && showLoading && (
        <span className="inline-block rounded-full bg-foreground px-4 py-2 text-sm text-background shadow-md motion-safe:animate-in motion-safe:fade-in">
          Loading page…
        </span>
      )}
    </div>
  )
}
