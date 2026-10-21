import { beforeEach, afterEach, describe, expect, it, vi } from "vitest"
import { IDBFactory, IDBKeyRange } from "fake-indexeddb"
import { Blob as NodeBlob } from "node:buffer"
import { loadDraft, saveDraft } from "./draft-addition"
import type { ResolvedScan } from "./draft-addition"
import type { EnvelopeMatchResult } from "@/generated/models"

// jsdom's Blob does not survive the in-memory IndexedDB's structured clone, so
// photos here are Node's Blob, as in the page review store's tests.
// SAFETY: Node's Blob reads the same as the DOM Blob a scan's photo is typed as.
const photo = () => new NodeBlob(["scan"]) as Blob

// A field a later API version adds: the Detection must keep it.
const later = { modelVersion: "v2" }
const detection: EnvelopeMatchResult = {
  data: { confident: true, candidates: [], ...later },
}

const scan = (cardId: string, extra: Partial<ResolvedScan> = {}) => ({
  cardId,
  quantity: 2,
  candidateIds: ["card-1", "card-2"],
  ...extra,
})

beforeEach(() => {
  // A fresh in-memory database for every test.
  vi.stubGlobal("indexedDB", new IDBFactory())
  vi.stubGlobal("IDBKeyRange", IDBKeyRange)
})
afterEach(() => {
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe("the stored Draft Addition", () => {
  it("keeps resolved Scans with their photos and Detections, per user and Collection", async () => {
    await saveDraft("user-1", "col-1", [
      scan("card-1", { photo: photo(), detection }),
    ])
    await saveDraft("user-1", "col-2", [scan("card-9", { candidateIds: [] })])

    const [kept] = await loadDraft("user-1", "col-1")
    expect(kept).toMatchObject({ cardId: "card-1", quantity: 2 })
    expect(await kept.photo?.text()).toBe("scan")
    expect(kept.detection).toEqual(detection)
    expect(await loadDraft("user-1", "col-2")).toEqual([
      { cardId: "card-9", quantity: 2, candidateIds: [] },
    ])
  })

  it("shows another user none of the saved Scans", async () => {
    await saveDraft("user-1", "col-1", [scan("card-1")])

    expect(await loadDraft("user-2", "col-1")).toEqual([])
  })

  it("moves a draft saved in local storage into the store without losing a Scan", async () => {
    localStorage.setItem(
      "cardstack:draft-addition:col-1",
      JSON.stringify([
        { cardId: "card-1", quantity: 1, candidateIds: [] },
        { cardId: "card-2", quantity: 3, candidateIds: ["card-2"] },
      ])
    )

    expect(await loadDraft("user-1", "col-1")).toEqual([
      { cardId: "card-1", quantity: 1, candidateIds: [] },
      { cardId: "card-2", quantity: 3, candidateIds: ["card-2"] },
    ])
    expect(localStorage.getItem("cardstack:draft-addition:col-1")).toBeNull()
    expect(await loadDraft("user-1", "col-1")).toHaveLength(2)
  })

  it("discards a draft stored in the old format, which had no alternatives", async () => {
    localStorage.setItem(
      "cardstack:draft-addition:col-1",
      JSON.stringify([{ cardId: "card-1", quantity: 1 }])
    )

    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })

  it("forgets the draft once it is saved empty", async () => {
    await saveDraft("user-1", "col-1", [scan("card-1")])
    await saveDraft("user-1", "col-1", [])

    expect(await loadDraft("user-1", "col-1")).toEqual([])
  })

  it("works from memory when IndexedDB is missing, and never throws", async () => {
    vi.stubGlobal("indexedDB", undefined)

    await expect(saveDraft("user-1", "col-1", [scan("card-1")])).resolves.toBe(
      undefined
    )
    await expect(loadDraft("user-1", "col-1")).resolves.toEqual([])
  })
})
