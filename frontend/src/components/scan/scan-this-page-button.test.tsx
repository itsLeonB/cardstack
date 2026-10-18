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
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { ScanThisPageButton } from "./scan-this-page-button"
import { loadPageReview, savePageReview } from "@/lib/page-review-store"
import type { PendingPageReview } from "@/lib/page-review-store"
import { useSession } from "@/lib/session"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({ useSession: vi.fn() }))

const signedInAs = (id: string) =>
  vi.mocked(useSession).mockReturnValue({
    user: { id, name: null, email: null },
    isAuthenticated: true,
    isLoading: false,
  })

const reviewFor = (userId: string, page = 1): PendingPageReview => ({
  userId,
  collectionId: "col-1",
  page,
  grid: { rows: 1, columns: 1 },
  cells: [{ photo: new Blob(["cell"]), match: { status: "failed" } }],
})

function renderButton(onScan = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <ScanThisPageButton collectionId="col-1" page={1} onScan={onScan} />
    </QueryClientProvider>
  )
  return onScan
}

beforeEach(() => {
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("ScanThisPageButton", () => {
  it("starts the scan straight away when the Page has no pending review", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    const onScan = renderButton()

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    await waitFor(() => expect(onScan).toHaveBeenCalledWith(undefined))
    expect(screen.queryByRole("alertdialog")).toBeNull()
  })

  it("asks Resume or start over, and Resume hands the review over", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1"))
    const onScan = renderButton()

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    const dialog = await screen.findByRole("alertdialog")
    expect(within(dialog).getByText("Resume or start over?")).toBeTruthy()

    await user.click(within(dialog).getByRole("button", { name: "Resume" }))
    expect(onScan).toHaveBeenCalledWith(
      expect.objectContaining({ userId: "user-1", page: 1 })
    )
  })

  it("discards the review only once Start over is chosen, then starts the scan", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1"))
    const onScan = renderButton()

    await user.click(screen.getByRole("button", { name: "Scan this Page" }))
    const dialog = await screen.findByRole("alertdialog")
    expect(
      await loadPageReview({ userId: "user-1", collectionId: "col-1", page: 1 })
    ).toBeDefined()
    expect(onScan).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole("button", { name: "Start over" }))
    await waitFor(() => expect(onScan).toHaveBeenCalledWith(undefined))
    expect(
      await loadPageReview({ userId: "user-1", collectionId: "col-1", page: 1 })
    ).toBeUndefined()
  })
})
