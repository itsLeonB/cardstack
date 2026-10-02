import { vi } from "vitest"

/**
 * jsdom does no layout: every box is 0x0 and there is no ResizeObserver, so
 * the virtual grid would render every row at 0px. This gives every element
 * the same width, every row `rowHeight`, and lets a test resize the
 * container with `resize` and scroll the window with `scrollTo`. Undo with `vi.restoreAllMocks()` and
 * `vi.unstubAllGlobals()`.
 */
export function stubGridLayout(width = 1000, rowHeight = 400) {
  let current = width
  let scrollY = 0
  const observers: ResizeObserverCallback[] = []
  vi.stubGlobal(
    "ResizeObserver",
    class {
      constructor(callback: ResizeObserverCallback) {
        observers.push(callback)
      }
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  )
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
    () => new DOMRect(0, -scrollY, current, 0)
  )
  vi.spyOn(window, "scrollY", "get").mockImplementation(() => scrollY)
  // The virtualizer measures a row by its offsetHeight.
  vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(
    rowHeight
  )
  // scrollToIndex clamps its target to the document height, which is 0 here.
  vi.spyOn(Element.prototype, "scrollHeight", "get").mockReturnValue(1_000_000)
  // The virtualizer scrolls the window to correct for measured row sizes.
  vi.spyOn(window, "scrollTo").mockImplementation(() => {})

  return {
    /** Scroll the window to `top` and let the virtualizer see it. Wrap in `act`. */
    scrollTo(top: number) {
      scrollY = top
      window.dispatchEvent(new Event("scroll"))
    },
    resize(next: number) {
      current = next
      // SAFETY: the grid's observer callbacks ignore their arguments.
      const observer = {} as ResizeObserver
      for (const observe of observers) observe([], observer)
    },
  }
}
