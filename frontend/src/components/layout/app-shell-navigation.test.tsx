import { act, cleanup, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Link,
  Outlet,
  RouterProvider,
} from "@tanstack/react-router"
import { afterEach, describe, expect, it } from "vitest"

import { NavigationLoading } from "./navigation-loading"

afterEach(cleanup)

function setup(loader: () => Promise<void>) {
  const root = createRootRoute({
    component: () => (
      <>
        <NavigationLoading />
        <Outlet />
      </>
    ),
  })
  const home = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: () => (
      <>
        <h1>Home</h1>
        <Link to="/catalog">Slow</Link>
        <Link
          to="/collections/$collectionId/scan"
          params={{ collectionId: "fixture" }}
        >
          Scan
        </Link>
        <Link to="/wishlist">Other</Link>
        <span role="status">Page status</span>
      </>
    ),
  })
  const slow = createRoute({
    getParentRoute: () => root,
    path: "/catalog",
    loader,
    component: () => <h1>Done</h1>,
    errorComponent: () => <h1>Failed</h1>,
  })
  const scan = createRoute({
    getParentRoute: () => root,
    path: "/collections/$collectionId/scan",
    loader,
    component: () => <h1>Scan</h1>,
  })
  const other = createRoute({
    getParentRoute: () => root,
    path: "/wishlist",
    component: () => <h1>Other</h1>,
  })
  const router = createRouter({
    routeTree: root.addChildren([home, slow, scan, other]),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  render(<RouterProvider router={router} />)
  return router
}

describe("navigation loading", () => {
  it("does not show feedback for a quick navigation", async () => {
    setup(() => Promise.resolve())
    const user = userEvent.setup()
    await user.click(await screen.findByRole("link", { name: "Slow" }))
    expect(await screen.findByRole("heading", { name: "Done" })).toBeTruthy()
    await new Promise((resolve) => setTimeout(resolve, 250))
    expect(screen.queryByText("Loading page…")).toBeNull()
  })

  it("shows pending work and removes it when the route completes", async () => {
    let resolve!: () => void
    setup(
      () =>
        new Promise<void>((done) => {
          resolve = done
        })
    )
    const user = userEvent.setup()
    await user.click(await screen.findByRole("link", { name: "Slow" }))
    const loading = await screen.findByText("Loading page…")
    expect(loading.closest('[aria-live="polite"]')).toBeTruthy()
    expect(screen.getAllByRole("status")).toHaveLength(1)
    await act(async () => resolve())
    expect(await screen.findByRole("heading", { name: "Done" })).toBeTruthy()
    await waitFor(() =>
      expect(screen.queryByText("Loading page…")).not.toBeTruthy()
    )
  })

  it("shows feedback while navigating to the scan route", async () => {
    let resolve!: () => void
    setup(
      () =>
        new Promise<void>((done) => {
          resolve = done
        })
    )
    const user = userEvent.setup()
    await user.click(await screen.findByRole("link", { name: "Scan" }))
    expect(await screen.findByText("Loading page…")).toBeTruthy()
    await act(async () => resolve())
    expect(await screen.findByRole("heading", { name: "Scan" })).toBeTruthy()
    await waitFor(() => expect(screen.queryByText("Loading page…")).toBeNull())
  })

  it("clears feedback when navigation fails or is replaced", async () => {
    let reject!: (error: Error) => void
    const router = setup(
      () =>
        new Promise<void>((_, fail) => {
          reject = fail
        })
    )
    const user = userEvent.setup()
    await user.click(await screen.findByRole("link", { name: "Slow" }))
    expect(await screen.findByText("Loading page…")).toBeTruthy()
    await act(async () => reject(new Error("failed")))
    expect(await screen.findByRole("heading", { name: "Failed" })).toBeTruthy()
    await waitFor(() =>
      expect(screen.queryByText("Loading page…")).not.toBeTruthy()
    )
    await act(async () => {
      await router.navigate({ to: "/" })
    })
    await user.click(await screen.findByRole("link", { name: "Slow" }))
    expect(await screen.findByText("Loading page…")).toBeTruthy()
    await act(async () => {
      await router.navigate({ to: "/wishlist" })
    })
    expect(await screen.findByRole("heading", { name: "Other" })).toBeTruthy()
    await waitFor(() =>
      expect(screen.queryByText("Loading page…")).not.toBeTruthy()
    )
  })
})
