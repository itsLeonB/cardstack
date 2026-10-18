import { beforeEach, describe, expect, it, vi } from "vitest"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
import {
  listPageReviews,
  loadPageReview,
  removePageReview,
  renamePageReview,
  savePageReview,
} from "./page-review-store"
import type { PendingPageReview } from "./page-review-store"

// jsdom's Blob does not survive the in-memory IndexedDB's structured clone, so
// photos here are Node's Blob, which does and is what a browser's Blob is to IndexedDB.
// SAFETY: Node's Blob reads the same as the DOM Blob that a cell's photo is typed as.
const photo = () => new NodeBlob(["cell"]) as Blob

const review = (page: number, collectionId = "col-1"): PendingPageReview => ({
  userId: "user-1",
  collectionId,
  page,
  grid: { rows: 1, columns: 2 },
  cells: [
    {
      photo: photo(),
      match: { status: "matched", confident: true, candidates: [] },
      cardId: "card-1",
    },
    {
      photo: photo(),
      match: { status: "failed" },
      mark: "empty",
    },
  ],
})

const key = (page: number, userId = "user-1", collectionId = "col-1") => ({
  userId,
  collectionId,
  page,
})

beforeEach(() => {
  // A fresh in-memory database for every test.
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})

describe("page review store", () => {
  it("saves a review and loads it back by user, Collection and Page", async () => {
    await savePageReview(review(3))

    const loaded = await loadPageReview(key(3))
    expect(loaded?.page).toBe(3)
    expect(loaded?.grid).toEqual({ rows: 1, columns: 2 })
    expect(loaded?.cells).toHaveLength(2)
    expect(loaded?.cells[0].photo).toBeInstanceOf(NodeBlob)
    expect(loaded?.cells[0].cardId).toBe("card-1")
    expect(loaded?.cells[1].mark).toBe("empty")
    expect(await loadPageReview(key(4))).toBeUndefined()
  })

  it("overwrites the review for the same key, so the last save wins", async () => {
    await savePageReview(review(3))
    await savePageReview({ ...review(3), cells: [] })

    expect((await loadPageReview(key(3)))?.cells).toEqual([])
  })

  it("keeps another user's review and another binder's review apart", async () => {
    await savePageReview(review(3))

    expect(await loadPageReview(key(3, "user-2"))).toBeUndefined()
    expect(await loadPageReview(key(3, "user-1", "col-2"))).toBeUndefined()
  })

  it("lists one binder's reviews for one user, in Page order", async () => {
    await savePageReview(review(5))
    await savePageReview(review(2))
    await savePageReview(review(2, "col-2"))
    await savePageReview({ ...review(9), userId: "user-2" })

    const listed = await listPageReviews("user-1", "col-1")
    expect(listed.map((r) => r.page)).toEqual([2, 5])
  })

  it("removes a review, and removing a missing one is not an error", async () => {
    await savePageReview(review(3))

    await removePageReview(key(3))
    await removePageReview(key(3))
    expect(await loadPageReview(key(3))).toBeUndefined()
  })

  it("moves a review to another Page number", async () => {
    await savePageReview(review(3))

    await renamePageReview(key(3), 4)
    expect(await loadPageReview(key(3))).toBeUndefined()
    expect((await loadPageReview(key(4)))?.cells).toHaveLength(2)
  })

  it("rejects when IndexedDB is missing, so the caller can fall back", async () => {
    vi.stubGlobal("indexedDB", undefined)

    await expect(savePageReview(review(3))).rejects.toThrow()
    await expect(loadPageReview(key(3))).rejects.toThrow()
  })
})
