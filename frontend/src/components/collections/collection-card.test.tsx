import { afterEach, describe, expect, it } from "vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { CollectionCard, cardCountLabel } from "./collection-card"

afterEach(cleanup)

const collection = {
  id: "col-1",
  title: "Vintage binder",
  description: "",
  maxCardCount: 0,
  cardCount: 37,
}

async function renderCard(overrides: Partial<typeof collection> = {}) {
  const root = createRootRoute()
  const list = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: () => <CollectionCard collection={{ ...collection, ...overrides }} />,
  })
  const detail = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId",
    component: () => null,
  })
  const edit = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId/edit",
    component: () => null,
  })
  const router = createRouter({
    routeTree: root.addChildren([list, detail, edit]),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByText("Vintage binder")
  return router
}

describe("cardCountLabel", () => {
  it.each([
    [0, "0 cards"],
    [1, "1 card"],
    [37, "37 cards"],
  ])("renders %i as %s", (count, label) => {
    expect(cardCountLabel(count)).toBe(label)
  })
})

describe("CollectionCard", () => {
  it("shows the card count", async () => {
    await renderCard({ cardCount: 1 })
    screen.getByText("1 card")
  })

  it("has one link to the detail page, named by the title only", async () => {
    await renderCard()
    const link = screen.getByRole("link", { name: "Vintage binder" })
    expect(link.getAttribute("href")).toBe("/collections/col-1")
    // Stretched link: the overlay is CSS on this very link, not a second element.
    expect(link.className).toContain("after:inset-0")
  })

  it("navigates to the detail page when the title link is activated", async () => {
    const router = await renderCard()
    fireEvent.click(screen.getByRole("link", { name: "Vintage binder" }))
    await waitFor(() => expect(router.state.location.pathname).toBe("/collections/col-1"))
  })

  it("keeps the edit action independent of the detail link", async () => {
    const router = await renderCard()
    fireEvent.click(screen.getByRole("link", { name: "Edit" }))
    await waitFor(() => expect(router.state.location.pathname).toBe("/collections/col-1/edit"))
  })

  it("does not navigate when the delete trigger is clicked", async () => {
    const router = await renderCard()
    fireEvent.click(screen.getByRole("button", { name: "Delete Vintage binder" }))
    await screen.findByRole("alertdialog")
    expect(router.state.location.pathname).toBe("/")
  })

  it("lifts the actions above the stretched overlay", async () => {
    await renderCard()
    const actions = screen.getByRole("link", { name: "Edit" }).closest("[data-slot=card-action]")
    expect(actions?.className).toContain("z-10")
  })
})
