import { afterEach, describe, expect, it } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router"
import { Breadcrumbs } from "./breadcrumbs"

afterEach(cleanup)

async function renderCrumbs() {
  const root = createRootRoute({
    component: () => (
      <Breadcrumbs
        crumbs={[
          { label: "Catalog", link: { to: "/catalog" } },
          { label: "Search" },
        ]}
      />
    ),
  })
  const router = createRouter({
    routeTree: root,
    history: createMemoryHistory({ initialEntries: ["/catalog/search"] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
}

describe("Breadcrumbs", () => {
  it("is a labelled navigation whose earlier crumbs link and last is the current page", async () => {
    await renderCrumbs()

    const nav = await screen.findByRole("navigation", { name: "Breadcrumb" })
    expect(nav).toBeTruthy()
    const ancestor = screen.getByRole("link", { name: "Catalog" })
    expect(ancestor.getAttribute("aria-current")).toBeNull()
    expect(screen.queryByRole("link", { name: "Search" })).toBeNull()
    expect(screen.getByText("Search").getAttribute("aria-current")).toBe("page")
    expect(screen.queryByRole("listitem")).toBeNull()
  })
})
