import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BinderView } from "./binder-view"
import { catalogCard, fakeBinder, stubBinderApi } from "@/test-binder-api"
import type { FakeBinder } from "@/test-binder-api"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function renderBinder(binder: FakeBinder) {
  stubBinderApi(binder)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <BinderView collectionId="col-1" />
    </QueryClientProvider>
  )
}

// Waits for the Page to load: until then the previous Page stays on screen,
// and its Slots carry the same names.
async function goToPage(page: number) {
  fireEvent.change(screen.getByRole("spinbutton", { name: "Page number" }), {
    target: { value: String(page) },
  })
  fireEvent.click(screen.getByRole("button", { name: "Go to page" }))
  await screen.findByText(`Page ${page} of 20`)
}

function moveRequests(binder: FakeBinder) {
  return binder.requests.filter((r) => r.path.endsWith("/binder/slots/move"))
}

const pikachu = catalogCard("pikachu", "Pikachu", "025")

describe("BinderView move and swap", () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn()
  })

  it("moves a card to an empty Slot on another Page", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    expect(screen.getByRole("status", { name: "Move a card" })).toBeTruthy()

    await goToPage(2)
    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 2: empty" })
    )

    await waitFor(() =>
      expect(screen.queryByRole("status", { name: "Move a card" })).toBeNull()
    )
    expect(moveRequests(binder)[0]?.body).toEqual({
      from: { page: 1, row: 1, column: 1 },
      to: { page: 2, row: 1, column: 2 },
    })
    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 2: Pikachu, Base Set",
      })
    ).toBeTruthy()
  })

  it("swaps two cards when the target Slot is filled, including an Unidentified one", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 2,
      filled: {
        1: { "1,1": { state: "card", card: pikachu } },
        2: { "1,2": { state: "unidentified" } },
      },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    await goToPage(2)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 2: Unidentified",
      })
    )

    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 2: Pikachu, Base Set",
      })
    ).toBeTruthy()
    await goToPage(1)
    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Unidentified",
      })
    ).toBeTruthy()
  })

  it("moves an Unidentified Slot within its own Page", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "unidentified" } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Unidentified",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    fireEvent.click(
      await screen.findByRole("button", { name: "Row 2, column 2: empty" })
    )

    expect(
      await screen.findByRole("button", {
        name: "Row 2, column 2: Unidentified",
      })
    ).toBeTruthy()
    expect(moveRequests(binder)[0]?.body).toEqual({
      from: { page: 1, row: 1, column: 1 },
      to: { page: 1, row: 2, column: 2 },
    })
  })

  it("cancels the move when the source Slot is tapped again, without a request", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    fireEvent.click(
      screen.getByRole("button", { name: "Row 1, column 1: Pikachu, Base Set" })
    )

    expect(screen.queryByRole("status", { name: "Move a card" })).toBeNull()
    expect(moveRequests(binder)).toHaveLength(0)
  })

  it("cancels the move from its Cancel button, without a request", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    fireEvent.click(screen.getByRole("button", { name: "Cancel move" }))

    expect(screen.queryByRole("status", { name: "Move a card" })).toBeNull()
    expect(moveRequests(binder)).toHaveLength(0)
  })

  it("cancels the move on Escape, without a request", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    fireEvent.keyDown(document, { key: "Escape" })

    expect(screen.queryByRole("status", { name: "Move a card" })).toBeNull()
    expect(moveRequests(binder)).toHaveLength(0)
  })

  it("moves focus to the move banner once a Slot is picked", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))

    expect(document.activeElement).toBe(
      screen.getByRole("status", { name: "Move a card" })
    )
  })

  it("sends one request when the target is tapped twice while the move is in flight", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    const target = screen.getByRole("button", {
      name: "Row 1, column 2: empty",
    })
    fireEvent.click(target)
    fireEvent.click(target)

    await waitFor(() =>
      expect(screen.queryByRole("status", { name: "Move a card" })).toBeNull()
    )
    expect(moveRequests(binder)).toHaveLength(1)
  })

  it("shows the refusal when the source Slot no longer holds a card", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    // Another writer clears the source after it was picked.
    delete binder.filled[1]
    fireEvent.click(
      screen.getByRole("button", { name: "Row 1, column 2: empty" })
    )

    expect(
      await screen.findByText("this Slot holds no card to move")
    ).toBeTruthy()
    expect(screen.getByRole("status", { name: "Move a card" })).toBeTruthy()
  })

  it("keeps the move open and shows the refusal when the write fails", async () => {
    const binder = fakeBinder({
      rows: 2,
      columns: 2,
      cardCount: 1,
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
      writeError: { status: 409, detail: "That Slot cannot take a card." },
    })
    renderBinder(binder)
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Move" }))
    fireEvent.click(
      screen.getByRole("button", { name: "Row 2, column 2: empty" })
    )

    expect(
      await screen.findByText("That Slot cannot take a card.")
    ).toBeTruthy()
    expect(screen.getByRole("status", { name: "Move a card" })).toBeTruthy()
  })
})
