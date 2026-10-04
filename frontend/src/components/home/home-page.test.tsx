import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { useUser } from "@clerk/react"
import type * as CollectionsModule from "@/generated/endpoints/collections/collections"
import type * as InventoryModule from "@/generated/endpoints/inventory/inventory"
import { useListCollections } from "@/generated/endpoints/collections/collections"
import { useListMasterInventory } from "@/generated/endpoints/inventory/inventory"
import { clerkUserResult } from "@/test-clerk"
import { HomePage } from "./home-page"

// Network boundary: the generated orval hooks have no service layer to inject,
// and Clerk's hooks need a ClerkProvider talking to Clerk's servers.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () => ({ useUser: vi.fn(), useClerk: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/collections/collections", async () => {
  const actual = await vi.importActual<typeof CollectionsModule>(
    "@/generated/endpoints/collections/collections"
  )
  return { ...actual, useListCollections: vi.fn() }
})
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", async () => {
  const actual = await vi.importActual<typeof InventoryModule>(
    "@/generated/endpoints/inventory/inventory"
  )
  return { ...actual, useListMasterInventory: vi.fn() }
})

const mockSession = vi.mocked(useUser)
const mockCollections = vi.mocked(useListCollections)
const mockInventory = vi.mocked(useListMasterInventory)

afterEach(cleanup)

function session(state: "loading" | "guest" | "signed-in") {
  mockSession.mockReturnValue(clerkUserResult(state))
}

function dashboardData(
  collections: { id: string; title: string }[],
  total: number
) {
  // SAFETY: partial hook results, see above.
  mockCollections.mockReturnValue({
    data: { status: 200, data: { data: collections } },
    isPending: false,
    isError: false,
  } as any)
  // SAFETY: partial hook results, see above.
  mockInventory.mockReturnValue({
    data: {
      status: 200,
      data: { data: [], meta: { total, page: 1, limit: 1 } },
    },
    isPending: false,
    isError: false,
  } as any)
}

async function renderHome() {
  const root = createRootRoute()
  const routeTree = root.addChildren(
    [
      "/",
      "/auth/register",
      "/catalog",
      "/catalog/search",
      "/inventory",
      "/collections",
      "/collections/new",
      "/collections/$collectionId",
    ].map((path) =>
      createRoute({
        getParentRoute: () => root,
        path,
        component: path === "/" ? HomePage : undefined,
      })
    )
  )
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe("HomePage session states", () => {
  it("shows a skeleton and no marketing copy while the session loads", async () => {
    session("loading")
    await renderHome()

    expect(await screen.findByRole("status", { name: "Loading" })).toBeTruthy()
    expect(screen.queryByRole("heading")).toBeNull()
    expect(screen.queryByText(/Track every card/)).toBeNull()
  })
})

describe("guest landing", () => {
  it("offers both actions, the three steps and a closing call to action", async () => {
    session("guest")
    await renderHome()

    expect(
      (await screen.findAllByRole("link", { name: "Create account" })).map(
        (a) => a.getAttribute("href")
      )
    ).toEqual(["/auth/register", "/auth/register"])
    expect(
      screen
        .getByRole("link", { name: "Browse the catalog" })
        .getAttribute("href")
    ).toBe("/catalog")

    const steps = within(screen.getByRole("list")).getAllByRole("heading", {
      level: 3,
    })
    expect(steps.map((h) => h.textContent)).toEqual([
      "Browse the Catalog",
      "Add Cards to Collections",
      "See your Master Inventory",
    ])
  })
})

describe("signed-in dashboard", () => {
  it("greets the user, lists Collections and shows the distinct card count", async () => {
    session("signed-in")
    dashboardData([{ id: "c1", title: "Trade binder" }], 42)
    await renderHome()

    expect(
      await screen.findByRole("heading", { level: 1, name: "Welcome back" })
    ).toBeTruthy()
    expect(screen.getByText("Signed in as Ada Lovelace")).toBeTruthy()
    expect(
      screen.getByRole("link", { name: "Trade binder" }).getAttribute("href")
    ).toBe("/collections/c1")
    expect(screen.getByRole("link", { name: "New collection" })).toBeTruthy()
    expect(screen.getByText("42")).toBeTruthy()
    expect(screen.getByText("distinct cards")).toBeTruthy()
    expect(screen.queryByText(/Track every card/)).toBeNull()
    // Only the total is read, so the list request asks for a single row.
    expect(mockInventory.mock.calls[0]?.[0]).toEqual({ page: 1, limit: 1 })
  })

  it("greets by email when the account has no name", async () => {
    mockSession.mockReturnValue(
      clerkUserResult("signed-in", {
        id: "user_2",
        fullName: null,
        primaryEmailAddress: { emailAddress: "grace@example.com" },
      })
    )
    dashboardData([], 0)
    await renderHome()

    expect(
      await screen.findByText("Signed in as grace@example.com")
    ).toBeTruthy()
  })

  it("explains what to do first when there are no Collections", async () => {
    session("signed-in")
    dashboardData([], 0)
    await renderHome()

    expect(
      await screen.findByText(/Start by creating a Collection/)
    ).toBeTruthy()
    expect(screen.getByRole("link", { name: "New collection" })).toBeTruthy()
  })
})

describe("signed-in dashboard load failures", () => {
  const failedQuery = (refetch: () => void) => ({
    data: undefined,
    isPending: false,
    isError: true,
    refetch,
  })

  it("shows an alert with a working Retry when Collections fail to load", async () => {
    session("signed-in")
    dashboardData([], 3)
    const refetch = vi.fn()
    // SAFETY: partial hook result, see session().
    mockCollections.mockReturnValue(failedQuery(refetch) as any)
    await renderHome()

    expect((await screen.findByRole("alert")).textContent).toBe(
      "Could not load your Collections."
    )
    expect(screen.queryByText(/Start by creating a Collection/)).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))
    expect(refetch).toHaveBeenCalledTimes(1)
  })

  it("shows an alert with a working Retry when the Master Inventory total fails to load", async () => {
    session("signed-in")
    dashboardData([{ id: "c1", title: "Trade binder" }], 0)
    const refetch = vi.fn()
    // SAFETY: partial hook result, see session().
    mockInventory.mockReturnValue(failedQuery(refetch) as any)
    await renderHome()

    expect((await screen.findByRole("alert")).textContent).toBe(
      "Could not load your Master Inventory total."
    )
    expect(screen.queryByText("distinct cards")).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Retry" }))
    expect(refetch).toHaveBeenCalledTimes(1)
  })

  it("labels the loading states", async () => {
    session("signed-in")
    // SAFETY: partial hook results, see session().
    mockCollections.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as any)
    // SAFETY: partial hook results, see session().
    mockInventory.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as any)
    await renderHome()

    expect(
      await screen.findByRole("status", { name: "Loading Collections" })
    ).toBeTruthy()
    expect(
      screen.getByRole("status", { name: "Loading Master Inventory total" })
    ).toBeTruthy()
  })

  it("lists five Collections and links to the rest", async () => {
    session("signed-in")
    dashboardData(
      Array.from({ length: 6 }, (_, i) => ({
        id: `c${i}`,
        title: `Binder ${i}`,
      })),
      1
    )
    await renderHome()

    expect(await screen.findByRole("link", { name: "Binder 4" })).toBeTruthy()
    expect(screen.queryByRole("link", { name: "Binder 5" })).toBeNull()
    expect(
      screen
        .getByRole("link", { name: "View all Collections" })
        .getAttribute("href")
    ).toBe("/collections")
  })
})
