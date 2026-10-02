import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router"
import { Breadcrumbs } from "./breadcrumbs"

afterEach(cleanup)

async function renderCrumbs(labels = ["Catalog", "Search"]) {
  const root = createRootRoute({
    component: () => (
      <Breadcrumbs
        crumbs={[
          { label: labels[0]!, link: { to: "/catalog" } },
          { label: labels[1]! },
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

  it("renders crumbs that share a label without key collisions", async () => {
    const errors: unknown[][] = []
    const spy = vi.spyOn(console, "error").mockImplementation((...args) => void errors.push(args))
    await renderCrumbs(["Edit", "Edit"])

    await screen.findByRole("navigation", { name: "Breadcrumb" })
    expect(errors.filter((args) => String(args[0]).includes("same key"))).toEqual([])
    spy.mockRestore()
  })
})
