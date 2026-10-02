import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { CrashFallback } from "./crash-fallback"

afterEach(cleanup)

const SECRET = "boom: secret internal detail"

function Exploder(): never {
  throw new Error(SECRET)
}

describe("CrashFallback", () => {
  it("replaces a thrown route error with a friendly message and no error text", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {})
    const root = createRootRoute()
    const router = createRouter({
      routeTree: root.addChildren([
        createRoute({ getParentRoute: () => root, path: "/", component: Exploder }),
      ]),
      history: createMemoryHistory({ initialEntries: ["/"] }),
      defaultErrorComponent: CrashFallback,
    })
    render(<RouterProvider router={router} />)

    expect(await screen.findByRole("heading", { name: "Something went wrong" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Reload" })).toBeTruthy()
    expect(document.body.textContent).not.toContain(SECRET)
    expect(document.body.textContent).not.toContain("Error:")
  })

  it("reloads the page from the Reload button", async () => {
    const reload = vi.fn()
    vi.stubGlobal("location", { ...window.location, reload })
    render(<CrashFallback />)

    fireEvent.click(screen.getByRole("button", { name: "Reload" }))

    expect(reload).toHaveBeenCalledOnce()
    vi.unstubAllGlobals()
  })
})
