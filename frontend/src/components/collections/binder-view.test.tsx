import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { toast } from "sonner"
import { BinderView, CreateBinderPanel } from "./binder-view"
import { NETWORK_ERROR } from "@/lib/collections"
import { catalogCard, fakeBinder, stubBinderApi } from "@/test-binder-api"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

function goToPage(page: number) {
  fireEvent.change(screen.getByRole("spinbutton", { name: "Page number" }), {
    target: { value: String(page) },
  })
  fireEvent.click(screen.getByRole("button", { name: "Go to page" }))
}

// The swipe handlers read `touches` and `changedTouches` off the event.
function swipe(
  target: HTMLElement,
  from: { x: number; y: number },
  to: { x: number; y: number }
) {
  const start = new Event("touchstart", { bubbles: true })
  Object.defineProperty(start, "touches", {
    value: [{ clientX: from.x, clientY: from.y }],
  })
  fireEvent(target, start)
  const end = new Event("touchend", { bubbles: true })
  Object.defineProperty(end, "changedTouches", {
    value: [{ clientX: to.x, clientY: to.y }],
  })
  fireEvent(target, end)
}

const pikachu = catalogCard("pikachu", "Pikachu", "025")

describe("BinderView", () => {
  it("shows the Page's Slots by state and the card total against capacity", async () => {
    stubBinderApi(
      fakeBinder({
        filled: {
          1: {
            "1,1": { state: "card", card: pikachu },
            "1,2": { state: "unidentified" },
          },
        },
      })
    )
    renderWithClient(<BinderView collectionId="col-1" />)

    expect(
      await screen.findByRole("button", { name: "Row 1, column 1: Pikachu" })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Row 1, column 2: Unidentified" })
    ).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Row 2, column 1: empty" })
    ).toBeTruthy()
    expect(screen.getByText("Page 1 of 20")).toBeTruthy()
    expect(
      screen.getByText("Cards and Unidentified: 2 of 80 slots")
    ).toBeTruthy()
    expect(screen.getByText("Unidentified: 1")).toBeTruthy()
  })

  it("swipes, steps and jumps between Pages", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    swipe(
      screen.getByRole("region", { name: "Binder page" }),
      { x: 200, y: 100 },
      { x: 100, y: 105 }
    )
    expect(await screen.findByText("Page 2 of 20")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))
    expect(await screen.findByText("Page 3 of 20")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Previous page" }))
    expect(await screen.findByText("Page 2 of 20")).toBeTruthy()

    goToPage(7)
    expect(await screen.findByText("Page 7 of 20")).toBeTruthy()
    expect(binder.requests.map((r) => r.path)).toContain(
      "/collections/col-1/binder/pages/7"
    )
  })

  it("moves two Pages on two quick Next taps", async () => {
    stubBinderApi(fakeBinder())
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    const next = screen.getByRole("button", { name: "Next page" })
    fireEvent.click(next)
    fireEvent.click(next)

    expect(await screen.findByText("Page 3 of 20")).toBeTruthy()
  })

  it("does nothing on Previous at the first Page or Next at the last", async () => {
    const binder = fakeBinder({ leafCount: 1 })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 2")

    fireEvent.click(screen.getByRole("button", { name: "Previous page" }))
    expect(screen.getByText("Page 1 of 2")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))
    expect(await screen.findByText("Page 2 of 2")).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Next page" }))
    expect(screen.getByText("Page 2 of 2")).toBeTruthy()
    expect(binder.requests.map((r) => r.path)).not.toContain(
      "/collections/col-1/binder/pages/3"
    )
  })

  it("ignores a short swipe and a mostly vertical one", async () => {
    stubBinderApi(fakeBinder())
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    const pageRegion = screen.getByRole("region", { name: "Binder page" })

    swipe(pageRegion, { x: 200, y: 100 }, { x: 170, y: 100 })
    swipe(pageRegion, { x: 200, y: 100 }, { x: 120, y: 300 })

    expect(screen.getByText("Page 1 of 20")).toBeTruthy()
  })

  it("adds a Leaf before the Leaf being viewed and turns to its front Page", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    goToPage(3)
    await screen.findByText("Page 3 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Add Leaf before" }))

    expect(await screen.findByText("Page 3 of 22")).toBeTruthy()
    expect(binder.requests).toContainEqual(
      expect.objectContaining({
        method: "POST",
        path: "/collections/col-1/binder/leaves",
        body: { leaf: 2, placement: "before" },
      })
    )
  })

  it("adds a Leaf after the Leaf being viewed and turns to its front Page", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    goToPage(3)
    await screen.findByText("Page 3 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Add Leaf after" }))

    expect(await screen.findByText("Page 5 of 22")).toBeTruthy()
    expect(binder.requests).toContainEqual(
      expect.objectContaining({
        method: "POST",
        path: "/collections/col-1/binder/leaves",
        body: { leaf: 2, placement: "after" },
      })
    )
  })

  it("removes the Leaf being viewed and turns to the Leaf that takes its place", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    goToPage(5)
    await screen.findByText("Page 5 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Remove Leaf" }))

    expect(await screen.findByText("Page 5 of 18")).toBeTruthy()
    expect(binder.requests).toContainEqual(
      expect.objectContaining({
        method: "DELETE",
        path: "/collections/col-1/binder/leaves/3",
      })
    )
  })

  it("shows the refusal naming the Page that holds a card", async () => {
    const binder = fakeBinder({
      filled: { 6: { "1,1": { state: "unidentified" } } },
    })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    goToPage(5)
    await screen.findByText("Page 5 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Remove Leaf" }))

    const alert = await screen.findByRole("alert")
    expect(alert.textContent).toBe(
      "page 6 holds a card; clear it before removing its Leaf"
    )
    expect(screen.getByText("Page 5 of 20")).toBeTruthy()
    expect(binder.leafCount).toBe(10)
  })

  it("reports a network failure on Remove binder as a toast", async () => {
    stubBinderApi(fakeBinder())
    const error = vi.spyOn(toast, "error")
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch")
      })
    )

    fireEvent.click(screen.getByRole("button", { name: "Remove binder" }))
    fireEvent.click(
      screen.getByRole("button", { name: "Confirm remove binder" })
    )

    await vi.waitFor(() => expect(error).toHaveBeenCalledWith(NETWORK_ERROR))
    error.mockRestore()
  })

  it("saves new rows and columns on an empty binder", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Change layout" }))
    fireEvent.change(screen.getByLabelText("Rows per Page"), {
      target: { value: "3" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Save layout" }))

    expect(
      await screen.findByRole("button", { name: "Row 3, column 1: empty" })
    ).toBeTruthy()
    expect(binder.requests).toContainEqual(
      expect.objectContaining({
        method: "PUT",
        path: "/collections/col-1/binder/layout",
        body: { rows: 3, columns: 2 },
      })
    )
  })

  it("offers the Layout edit and removal only while every Slot is empty", async () => {
    stubBinderApi(fakeBinder())
    const { unmount } = renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    expect(screen.getByRole("button", { name: "Change layout" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Remove binder" })).toBeTruthy()
    unmount()

    stubBinderApi(
      fakeBinder({ filled: { 1: { "1,1": { state: "unidentified" } } } })
    )
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    expect(screen.queryByRole("button", { name: "Change layout" })).toBeNull()
    expect(screen.queryByRole("button", { name: "Remove binder" })).toBeNull()
  })

  it("removes the binder's Layout after confirming, without refetching its Pages", async () => {
    const binder = fakeBinder()
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")

    fireEvent.click(screen.getByRole("button", { name: "Remove binder" }))
    fireEvent.click(
      screen.getByRole("button", { name: "Confirm remove binder" })
    )
    await vi.waitFor(() =>
      expect(binder.requests).toContainEqual(
        expect.objectContaining({
          method: "DELETE",
          path: "/collections/col-1/binder",
        })
      )
    )
    const pageReads = () =>
      binder.requests.filter((r) => r.path.includes("/binder/pages/")).length
    const readsAfterRemove = pageReads()
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(pageReads()).toBe(readsAfterRemove)
  })
})

describe("CreateBinderPanel", () => {
  it("creates a binder with the chosen preset and the default 10 Leaves", async () => {
    const binder = fakeBinder({ exists: false })
    stubBinderApi(binder)
    renderWithClient(<CreateBinderPanel collectionId="col-1" />)

    fireEvent.change(screen.getByRole("combobox", { name: "Layout" }), {
      target: { value: "3x4" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Create binder" }))

    await vi.waitFor(() =>
      expect(binder.requests).toContainEqual(
        expect.objectContaining({
          method: "POST",
          path: "/collections/col-1/binder",
          body: { rows: 3, columns: 4, leafCount: 10 },
        })
      )
    )
  })

  it("refuses a Leaf count above 100 without sending a request", () => {
    const binder = fakeBinder({ exists: false })
    stubBinderApi(binder)
    renderWithClient(<CreateBinderPanel collectionId="col-1" />)

    fireEvent.change(screen.getByRole("spinbutton", { name: "Leaves" }), {
      target: { value: "101" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Create binder" }))

    expect(
      screen.getByText("A binder holds from 1 to 100 Leaves.")
    ).toBeTruthy()
    expect(binder.requests).toHaveLength(0)
  })
})
