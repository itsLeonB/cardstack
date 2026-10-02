import { afterEach, describe, expect, it } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { NotFound, notFoundResource } from "./not-found"

afterEach(cleanup)

function renderAt(path: string) {
  const root = createRootRoute({
    notFoundComponent: NotFound,
    component: () => (
      <main>
        <Outlet />
      </main>
    ),
  })
  const missing = createRoute({
    getParentRoute: () => root,
    path: "/collections/$id",
    loader: () => {
      throw notFoundResource("Collection")
    },
  })
  const router = createRouter({
    routeTree: root.addChildren([missing]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  render(<RouterProvider router={router} />)
}

describe("NotFound", () => {
  it("offers Home and Catalog for an unknown URL", async () => {
    renderAt("/nope")

    expect(await screen.findByRole("heading", { name: "Page not found" })).toBeTruthy()
    expect(screen.getByRole("link", { name: "Home" }).getAttribute("href")).toBe("/")
    expect(screen.getByRole("link", { name: "Catalog" }).getAttribute("href")).toBe("/catalog")
  })

  it("names the missing entity when a loader throws notFoundResource", async () => {
    renderAt("/collections/abc")

    expect(await screen.findByRole("heading", { name: "Collection not found" })).toBeTruthy()
    expect(screen.getByRole("link", { name: "Home" })).toBeTruthy()
  })
})
