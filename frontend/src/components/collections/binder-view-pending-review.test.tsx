import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
import { toast } from "sonner"
import { BinderView } from "./binder-view"
import { savePageReview } from "@/lib/page-review-store"
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

beforeEach(() => {
  // A fresh in-memory database for every test.
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

// SAFETY: Node's Blob reads the same as the DOM Blob that a cell's photo is typed as.
const photo = () => new NodeBlob(["cell"]) as Blob

// A pending review on the default binder (2 rows by 2 columns, so 4 cells).
function pendingReview(page: number, cells = 4): PendingPageReview {
  return {
    userId: "user-1",
    collectionId: "col-1",
    page,
    grid: { rows: 1, columns: cells },
    cells: Array.from({ length: cells }, () => ({
      photo: photo(),
      match: { status: "failed" as const },
      mark: "empty" as const,
    })),
  }
}

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

const BANNER = "This Page has an unfinished scan."

describe("BinderView pending Page Reviews follow their Leaf", () => {
  it("moves a pending review to its new Page when a Leaf is added before it", async () => {
    stubBinderApi(fakeBinder())
    await savePageReview(pendingReview(4))
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 20")
    goToPage(4)
    expect(await screen.findByText(BANNER)).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Add Leaf before" }))
    expect(await screen.findByText("Page 3 of 22")).toBeTruthy()

    goToPage(6)
    expect(await screen.findByText(BANNER)).toBeTruthy()
    goToPage(4)
    expect(await screen.findByText("Page 4 of 22")).toBeTruthy()
    await waitFor(() => expect(screen.queryByText(BANNER)).toBeNull())
  })

  it("discards the review of a removed Leaf's own Pages with a one-line message", async () => {
    stubBinderApi(fakeBinder({ leafCount: 4 }))
    await savePageReview(pendingReview(3))
    const info = vi.spyOn(toast, "info")
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 8")
    goToPage(3)
    expect(await screen.findByText(BANNER)).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: "Remove Leaf" }))

    expect(await screen.findByText("Page 3 of 6")).toBeTruthy()
    await waitFor(() =>
      expect(info).toHaveBeenCalledWith(
        "Discarded the pending review of Page 3 with its Leaf."
      )
    )
    await waitFor(() => expect(screen.queryByText(BANNER)).toBeNull())
  })

  it("moves a review after a removed Leaf down to its new Page", async () => {
    stubBinderApi(fakeBinder({ leafCount: 4 }))
    await savePageReview(pendingReview(5))
    renderWithClient(<BinderView collectionId="col-1" />)
    await screen.findByText("Page 1 of 8")
    goToPage(3)
    await screen.findByText("Page 3 of 8")
    expect(screen.queryByText(BANNER)).toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "Remove Leaf" }))

    expect(await screen.findByText("Page 3 of 6")).toBeTruthy()
    expect(await screen.findByText(BANNER)).toBeTruthy()
  })
})

describe("BinderView discards a review whose Layout no longer matches", () => {
  it("discards it with a one-line message when the binder opens", async () => {
    stubBinderApi(fakeBinder())
    // Three cells on a 2 by 2 Layout.
    await savePageReview(pendingReview(1, 3))
    const info = vi.spyOn(toast, "info")
    renderWithClient(<BinderView collectionId="col-1" />)

    await waitFor(() =>
      expect(info).toHaveBeenCalledWith(
        "Discarded the pending review of Page 1: the Layout changed."
      )
    )
    expect(await screen.findByText("Page 1 of 20")).toBeTruthy()
    await waitFor(() => expect(screen.queryByText(BANNER)).toBeNull())
  })
})
