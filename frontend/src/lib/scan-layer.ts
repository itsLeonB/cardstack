import { createContext, useContext, useLayoutEffect } from "react"

/** Set by `AppShell`: tells it a full-screen scan layer is open, so it hides the mobile bar. */
export const ScanLayerContext = createContext<(active: boolean) => void>(
  () => {}
)

/** Marks the caller as a full-screen scan layer for as long as it is mounted. */
export function useScanLayer() {
  const setScanLayer = useContext(ScanLayerContext)
  // Before paint: the bar must never show over the layer for a frame.
  useLayoutEffect(() => {
    setScanLayer(true)
    return () => setScanLayer(false)
  }, [setScanLayer])
}
