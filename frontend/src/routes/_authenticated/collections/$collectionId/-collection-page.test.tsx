import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { NotFound } from "@/components/layout/not-found"
import { catalogCard, fakeBinder, stubBinderApi } from "@/test-binder-api"
import type { FakeBinder } from "@/test-binder-api"
import { Route as CollectionRoute } from "./index"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

// The real Collection page in a memory router, over the faked binder API.
async function renderCollectionPage(binder: FakeBinder) {
  stubBinderApi(binder)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const collection = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId/",
    ...pick(CollectionRoute.options),
  })
  const router = createRouter({
    routeTree: root.addChildren([collection]),
    history: createMemoryHistory({
      initialEntries: ["/collections/col-1/"],
    }),
    context: { queryClient },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  // The page's title shows once its Collection has loaded.
  await screen.findByRole("heading", { name: "Binder", level: 1 })
}

function pick(options: {
  loader?: unknown
  component?: unknown
  validateSearch?: unknown
}): any {
  const { loader, component, validateSearch } = options
  return { loader, component, validateSearch }
}

const pikachu = catalogCard("pikachu", "Pikachu", "025")

describe("Collection page", () => {
  it("lists each binder Card's Slot positions, one per copy", async () => {
    await renderCollectionPage(
      fakeBinder({
        rows: 2,
        columns: 3,
        entries: [
          {
            card: pikachu,
            quantity: 2,
            positions: [
              { page: 4, row: 2, column: 3 },
              { page: 1, row: 1, column: 1 },
            ],
          },
        ],
      })
    )

    expect(
      await screen.findByRole("button", { name: "Page 4 · row 2 · col 3" })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Page 1 · row 1 · col 1" })
    ).toBeTruthy()
  })

  it("opens the binder on the tapped position's Page with that Slot highlighted", async () => {
    Element.prototype.scrollIntoView = vi.fn()
    await renderCollectionPage(
      fakeBinder({
        rows: 2,
        columns: 3,
        filled: { 4: { "2,3": { state: "card", card: pikachu } } },
        entries: [
          {
            card: pikachu,
            quantity: 1,
            positions: [{ page: 4, row: 2, column: 3 }],
          },
        ],
      })
    )

    fireEvent.click(
      await screen.findByRole("button", { name: "Page 4 · row 2 · col 3" })
    )

    expect(await screen.findByText("Page 4 of 20")).toBeTruthy()
    const tile = screen.getByRole("button", {
      name: "Row 2, column 3: Pikachu",
    })
    expect(tile.getAttribute("aria-pressed")).toBe("true")
    expect(screen.queryByRole("region", { name: "Slot actions" })).toBeNull()
  })

  it("reloads the entries after a Leaf is added, so positions follow the renumbered Pages", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      entries: [
        {
          card: pikachu,
          quantity: 1,
          positions: [{ page: 1, row: 1, column: 1 }],
        },
      ],
    })
    await renderCollectionPage(binder)
    await screen.findByRole("button", { name: "Page 1 · row 1 · col 1" })
    const entryLoads = () =>
      binder.requests.filter((r) => r.path.endsWith("/entries")).length
    const before = entryLoads()

    fireEvent.click(screen.getByRole("button", { name: "Add Leaf after" }))

    await waitFor(() => expect(entryLoads()).toBeGreaterThan(before))
  })

  it("offers the Layout form on a plain Collection with no Cards", async () => {
    await renderCollectionPage(fakeBinder({ exists: false, cardCount: 0 }))

    expect(
      screen.getByRole("heading", { name: "Make this a binder" })
    ).toBeTruthy()
  })

  it("does not offer the Layout form on a plain Collection that holds Cards, and keeps its links", async () => {
    await renderCollectionPage(fakeBinder({ exists: false, cardCount: 3 }))

    expect(
      screen.queryByRole("heading", { name: "Make this a binder" })
    ).toBeNull()
    expect(screen.getByRole("link", { name: "Scan cards" })).toBeTruthy()
    expect(screen.getByRole("link", { name: "Add Cards" })).toBeTruthy()
  })

  it("shows the binder view and hides the Scan and Add links on a binder", async () => {
    await renderCollectionPage(fakeBinder())

    expect(await screen.findByText("Page 1 of 20")).toBeTruthy()
    expect(screen.queryByRole("link", { name: "Scan cards" })).toBeNull()
    expect(screen.queryByRole("link", { name: "Add Cards" })).toBeNull()
    expect(
      screen.queryByRole("heading", { name: "Make this a binder" })
    ).toBeNull()
  })

  it("lists a binder's Cards without quantity controls", async () => {
    const entries = [
      { quantity: 2, card: catalogCard("pikachu", "Pikachu", "025") },
    ]
    await renderCollectionPage(fakeBinder({ entries }))

    expect(await screen.findByText("Pikachu")).toBeTruthy()
    expect(screen.queryByLabelText("Quantity of Pikachu")).toBeNull()
  })

  it("keeps quantity controls on a plain Collection's list", async () => {
    const entries = [
      { quantity: 2, card: catalogCard("pikachu", "Pikachu", "025") },
    ]
    await renderCollectionPage(
      fakeBinder({ exists: false, cardCount: 2, entries })
    )

    expect(await screen.findByLabelText("Quantity of Pikachu")).toBeTruthy()
  })

  it("shows the binder view once a binder is created from the plain page", async () => {
    const binder = fakeBinder({ exists: false, cardCount: 0 })
    await renderCollectionPage(binder)

    fireEvent.click(screen.getByRole("button", { name: "Create binder" }))

    expect(await screen.findByText("Page 1 of 20")).toBeTruthy()
    expect(
      screen.queryByRole("heading", { name: "Make this a binder" })
    ).toBeNull()
  })
})
