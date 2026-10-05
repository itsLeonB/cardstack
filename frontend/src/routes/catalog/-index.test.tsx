import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { Route as CatalogRoute } from "./index"

// The catalog browse page's Series headings: the real route in a memory
// router with `fetch` standing in for the API.
afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

const LOGO = "https://img.example.com/series/scarlet-violet.abc12345.webp"

async function visit(seriesList: { name: string; imageUrl: string }[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            data: {
              series: seriesList.map((entry, i) => ({
                ...entry,
                id: `sr${i}`,
                code: `c${i}`,
                expansionSets: [],
              })),
              ungroupedExpansionSets: [],
            },
          }),
          { status: 200, headers: { "content-type": "application/json" } }
        )
    )
  )
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRoute({ component: () => <Outlet /> })
  const routeTree = root.addChildren([
    createRoute({
      getParentRoute: () => root,
      path: "/catalog",
      component: CatalogRoute.options.component,
    }),
  ])
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/catalog"] }),
  })
  const { container } = render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole("region", { name: seriesList[0].name })
  return container
}

describe("catalog Series heading", () => {
  it("shows the logo as a decorative image, keeping the name as the section's accessible name", async () => {
    const container = await visit([
      { name: "Scarlet & Violet", imageUrl: LOGO },
    ])

    const heading = screen.getByRole("heading", {
      level: 2,
      name: "Scarlet & Violet",
    })
    const img = heading.querySelector("img")
    expect(img?.getAttribute("alt")).toBe("")
    // The hosted address as-is: no `/cdn-cgi/image/` rewrite.
    expect(img?.getAttribute("src")).toBe(LOGO)
    expect(container.innerHTML).not.toContain("cdn-cgi")
    expect(img?.getAttribute("height")).toBe("40")
    screen.getByRole("region", { name: "Scarlet & Violet" })
  })

  it("renders the plain text heading when the Series has no logo", async () => {
    const container = await visit([{ name: "Scarlet & Violet", imageUrl: "" }])

    const heading = screen.getByRole("heading", {
      level: 2,
      name: "Scarlet & Violet",
    })
    expect(heading.textContent).toBe("Scarlet & Violet")
    expect(container.querySelector("img")).toBeNull()
    screen.getByRole("region", { name: "Scarlet & Violet" })
  })

  it("falls back to the text heading when the logo fails to load", async () => {
    const container = await visit([
      { name: "Scarlet & Violet", imageUrl: LOGO },
    ])

    fireEvent.error(container.querySelector("img")!)

    const heading = screen.getByRole("heading", {
      level: 2,
      name: "Scarlet & Violet",
    })
    expect(heading.textContent).toBe("Scarlet & Violet")
    expect(container.querySelector("img")).toBeNull()
    screen.getByRole("region", { name: "Scarlet & Violet" })
  })
})
