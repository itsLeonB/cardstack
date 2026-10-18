import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { BinderView } from "./binder-view"
import { loadPageReview, savePageReview } from "@/lib/page-review-store"
import type { PendingPageReview } from "@/lib/page-review-store"
import { fakeBinder, stubBinderApi } from "@/test-binder-api"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({
  useSession: () => ({
    user: { id: "user-1", name: null, email: null },
    isAuthenticated: true,
    isLoading: false,
  }),
}))

const reviewFor = (page: number): PendingPageReview => ({
  userId: "user-1",
  collectionId: "col-1",
  page,
  grid: { rows: 2, columns: 2 },
  cells: [{ photo: new Blob(["cell"]), match: { status: "failed" } }],
})

function renderBinder() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  // The scan screen's leave blocker needs a router, as in the app.
  const root = createRootRoute()
  const binder = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: () => <BinderView collectionId="col-1" />,
  })
  const router = createRouter({
    routeTree: root.addChildren([binder]),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
  // Photos come back from fake IndexedDB as plain objects, which jsdom cannot turn into object URLs.
  URL.createObjectURL = vi.fn(() => "blob:photo")
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("BinderView pending Page reviews", () => {
  it("lists the pending Pages under the jump form, and a number jumps to that Page", async () => {
    const user = userEvent.setup()
    stubBinderApi(fakeBinder())
    await savePageReview(reviewFor(4))
    renderBinder()
    await screen.findByText("Page 1 of 20")

    const notice = await screen.findByRole("group", { name: "Pending scans" })
    expect(notice.textContent).toBe("Page 4 has a pending scan")

    await user.click(
      within(notice).getByRole("button", { name: "Go to Page 4" })
    )
    expect(await screen.findByText("Page 4 of 20")).toBeTruthy()
  })

  it("drops the notice once the Page's review is discarded from its banner", async () => {
    const user = userEvent.setup()
    stubBinderApi(fakeBinder())
    await savePageReview(reviewFor(1))
    renderBinder()

    await screen.findByRole("group", { name: "Pending scans" })
    await user.click(screen.getByRole("button", { name: "Discard" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Discard" }))

    await waitFor(() =>
      expect(screen.queryByRole("group", { name: "Pending scans" })).toBeNull()
    )
  })

  it("Start over discards the review and opens the photo step", async () => {
    const user = userEvent.setup()
    stubBinderApi(fakeBinder())
    await savePageReview(reviewFor(1))
    renderBinder()
    await screen.findByText("Page 1 of 20")

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Start over" }))

    expect(
      await screen.findByRole("heading", { name: "Scan Page 1" })
    ).toBeTruthy()
    expect(screen.queryByText("Check each cell.")).toBeNull()
  })

  it("Resume opens the pending review's cells", async () => {
    const user = userEvent.setup()
    stubBinderApi(fakeBinder())
    await savePageReview(reviewFor(1))
    renderBinder()
    await screen.findByText("Page 1 of 20")

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Resume" }))

    expect(await screen.findByText("Check each cell.")).toBeTruthy()
  })

  it("drops the notice once Confirm saves the Page's review", async () => {
    const user = userEvent.setup()
    stubBinderApi(fakeBinder())
    await savePageReview(reviewFor(1))
    renderBinder()
    await screen.findByText("Page 1 of 20")
    await screen.findByRole("group", { name: "Pending scans" })

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Resume" }))
    await user.click(await screen.findByRole("button", { name: "Confirm" }))

    await waitFor(() =>
      expect(screen.queryByRole("group", { name: "Pending scans" })).toBeNull()
    )
    expect(
      await loadPageReview({ userId: "user-1", collectionId: "col-1", page: 1 })
    ).toBeUndefined()
  })
})
