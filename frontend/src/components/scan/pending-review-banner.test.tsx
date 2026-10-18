import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { PendingReviewBanner } from "./pending-review-banner"
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

beforeEach(() => {
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("PendingReviewBanner", () => {
  it("offers Resume for the signed-in user's review, and hands the review over", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1"))
    const onResume = vi.fn()
    render(
      <PendingReviewBanner collectionId="col-1" page={1} onResume={onResume} />
    )

    await user.click(await screen.findByRole("button", { name: "Resume" }))
    expect(onResume).toHaveBeenCalledWith(
      expect.objectContaining({ userId: "user-1", page: 1 })
    )
  })

  it("shows nothing for another user's review on the same Page", async () => {
    signedInAs("user-1")
    await savePageReview(reviewFor("user-2"))
    render(
      <PendingReviewBanner collectionId="col-1" page={1} onResume={vi.fn()} />
    )

    // The read is asynchronous, so let it finish before asserting nothing shows.
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(screen.queryByRole("button", { name: "Resume" })).toBeNull()
  })

  it("removes the review after the user confirms Discard", async () => {
    const user = userEvent.setup()
    signedInAs("user-1")
    await savePageReview(reviewFor("user-1"))
    render(
      <PendingReviewBanner collectionId="col-1" page={1} onResume={vi.fn()} />
    )

    await user.click(await screen.findByRole("button", { name: "Discard" }))
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "Discard" }))

    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Resume" })).toBeNull()
    )
    expect(
      await loadPageReview({ userId: "user-1", collectionId: "col-1", page: 1 })
    ).toBeUndefined()
  })
})
