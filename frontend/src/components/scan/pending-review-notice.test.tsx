import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { PendingReviewNotice } from "./pending-review-notice"
import { savePageReview } from "@/lib/page-review-store"
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

const reviewFor = (
  userId: string,
  collectionId: string,
  page: number
): PendingPageReview => ({
  userId,
  collectionId,
  page,
  grid: { rows: 1, columns: 1 },
  cells: [{ photo: new Blob(["cell"]), match: { status: "failed" } }],
})

function renderNotice(onJump = vi.fn()) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <PendingReviewNotice collectionId="col-1" onJump={onJump} />
    </QueryClientProvider>
  )
  return onJump
}

beforeEach(() => {
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("PendingReviewNotice", () => {
  it("shows nothing when this binder has no pending review", async () => {
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1", "col-2", 3))
    renderNotice()

    // Let the list read finish before asserting that nothing is shown.
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(screen.queryByRole("group", { name: "Pending scans" })).toBeNull()
  })

  it("lists this binder's pending Pages in order, for the signed-in user only", async () => {
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1", "col-1", 7))
    await savePageReview(reviewFor("user-1", "col-1", 4))
    await savePageReview(reviewFor("user-1", "col-2", 2))
    await savePageReview(reviewFor("user-2", "col-1", 3))
    renderNotice()

    const notice = await screen.findByRole("group", { name: "Pending scans" })
    expect(notice.textContent).toBe("Pages 4, 7 have a pending scan")
  })

  it("uses the singular for one pending Page", async () => {
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1", "col-1", 4))
    renderNotice()

    const notice = await screen.findByRole("group", { name: "Pending scans" })
    expect(notice.textContent).toBe("Page 4 has a pending scan")
  })

  it("calls onJump with the Page whose number is pressed", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1", "col-1", 4))
    await savePageReview(reviewFor("user-1", "col-1", 7))
    const onJump = renderNotice()

    await user.click(
      await screen.findByRole("button", { name: "Go to Page 7" })
    )
    expect(onJump).toHaveBeenCalledWith(7)
  })
})
