import { describe, expect, it } from "vitest"
import { MatchQueue } from "./match-queue"

// Each run waits until the test settles it, so the test controls which slots are busy.
function harness(limit = 3) {
  const started: string[] = []
  const settle = new Map<string, () => void>()
  const signals = new Map<string, AbortSignal>()
  const queue = new MatchQueue<string>(limit, (id, _item, signal) => {
    started.push(id)
    signals.set(id, signal)
    return new Promise<void>((resolve) => settle.set(id, resolve))
  })
  return { queue, started, settle, signals }
}

describe("MatchQueue", () => {
  it("runs at most three matches at once, in the order they were added", () => {
    const { queue, started } = harness()
    for (const id of ["a", "b", "c", "d", "e"]) queue.add(id, id)

    expect(started).toEqual(["a", "b", "c"])
  })

  it("starts the next waiting match as soon as a slot frees", async () => {
    const { queue, started, settle } = harness()
    for (const id of ["a", "b", "c", "d"]) queue.add(id, id)

    settle.get("b")!()
    await Promise.resolve()
    await Promise.resolve()

    expect(started).toEqual(["a", "b", "c", "d"])
  })

  it("drops a waiting match on cancel without running it", () => {
    const { queue, started } = harness()
    for (const id of ["a", "b", "c", "d"]) queue.add(id, id)

    queue.cancel("d")

    expect(started).toEqual(["a", "b", "c"])
  })

  it("aborts a running match on cancel and hands its slot to the next one", () => {
    const { queue, started, signals } = harness()
    for (const id of ["a", "b", "c", "d"]) queue.add(id, id)

    queue.cancel("a")

    expect(signals.get("a")!.aborted).toBe(true)
    expect(started).toEqual(["a", "b", "c", "d"])
  })

  it("aborts everything on cancelAll without starting a waiting match", () => {
    const { queue, started, signals } = harness()
    for (const id of ["a", "b", "c", "d", "e"]) queue.add(id, id)

    queue.cancelAll()

    expect(["a", "b", "c"].map((id) => signals.get(id)!.aborted)).toEqual([
      true,
      true,
      true,
    ])
    expect(started).toEqual(["a", "b", "c"])
  })

  it("keeps going after a match fails", async () => {
    const started: string[] = []
    const queue = new MatchQueue<string>(1, async (id) => {
      started.push(id)
      if (id === "a") throw new Error("boom")
    })

    queue.add("a", "a")
    queue.add("b", "b")
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(started).toEqual(["a", "b"])
  })
})
