import { afterEach, describe, expect, it } from "vitest"
import {
  addOne,
  clearDraft,
  lowerOne,
  loadDraft,
  removeRow,
  saveDraft,
} from "./draft-addition"

afterEach(() => localStorage.clear())

describe("draft rows", () => {
  it("merges the same Card into one row", () => {
    const rows = addOne(addOne([], "a"), "a")
    expect(rows).toEqual([{ cardId: "a", quantity: 2 }])
  })

  it("keeps different Cards in scan order", () => {
    expect(addOne(addOne([], "a"), "b").map((r) => r.cardId)).toEqual([
      "a",
      "b",
    ])
  })

  it("lowers a quantity but never below 1, and removes a row", () => {
    const rows = addOne(addOne([], "a"), "a")
    expect(lowerOne(rows, "a")).toEqual([{ cardId: "a", quantity: 1 }])
    expect(lowerOne(lowerOne(rows, "a"), "a")).toEqual([
      { cardId: "a", quantity: 1 },
    ])
    expect(removeRow(rows, "a")).toEqual([])
  })
})

describe("draft storage", () => {
  it("restores each Collection's own draft", () => {
    saveDraft("c1", [{ cardId: "a", quantity: 2 }])
    saveDraft("c2", [{ cardId: "b", quantity: 1 }])
    expect(loadDraft("c1")).toEqual([{ cardId: "a", quantity: 2 }])
    expect(loadDraft("c2")).toEqual([{ cardId: "b", quantity: 1 }])
  })

  it("stores only ids and quantities", () => {
    saveDraft("c1", [{ cardId: "a", quantity: 2 }])
    const raw = Object.values(localStorage).join()
    expect(JSON.parse(raw)).toEqual([{ cardId: "a", quantity: 2 }])
  })

  it("drops a draft on clear and when saved empty", () => {
    saveDraft("c1", [{ cardId: "a", quantity: 1 }])
    clearDraft("c1")
    expect(loadDraft("c1")).toEqual([])
    saveDraft("c2", [])
    expect(localStorage.length).toBe(0)
  })

  it("ignores corrupt stored data", () => {
    saveDraft("c1", [{ cardId: "a", quantity: 1 }])
    const key = localStorage.key(0)!
    localStorage.setItem(key, "{not json")
    expect(loadDraft("c1")).toEqual([])
    localStorage.setItem(key, JSON.stringify([{ cardId: "a", quantity: -3 }]))
    expect(loadDraft("c1")).toEqual([])
  })
})
