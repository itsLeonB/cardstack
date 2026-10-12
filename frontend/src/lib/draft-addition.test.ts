import { afterEach, describe, expect, it } from "vitest"
import { clearDraft, loadDraft, saveDraft } from "./draft-addition"
import type { ResolvedScan } from "./draft-addition"

afterEach(() => localStorage.clear())

const scan: ResolvedScan = {
  cardId: "card-1",
  quantity: 2,
  candidateIds: ["card-1", "card-2"],
}

describe("the stored Draft Addition", () => {
  it("round-trips resolved Scans per Collection", () => {
    saveDraft("col-1", [scan])
    saveDraft("col-2", [{ ...scan, cardId: "card-9", candidateIds: [] }])

    expect(loadDraft("col-1")).toEqual([scan])
    expect(loadDraft("col-2")).toEqual([
      { cardId: "card-9", quantity: 2, candidateIds: [] },
    ])
  })

  it("discards a draft stored in the old format, which had no alternatives", () => {
    localStorage.setItem(
      "cardstack:draft-addition:col-1",
      JSON.stringify([{ cardId: "card-1", quantity: 1 }])
    )

    expect(loadDraft("col-1")).toEqual([])
  })

  it("forgets the draft once it is empty or cleared", () => {
    saveDraft("col-1", [scan])
    clearDraft("col-1")

    expect(loadDraft("col-1")).toEqual([])
    expect(localStorage.getItem("cardstack:draft-addition:col-1")).toBeNull()
  })
})
