import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type * as TanStackRouter from "@tanstack/react-router"
import { BinderView } from "./binder-view"
import { getGetCollectionQueryKey } from "@/generated/endpoints/collections/collections"
import {
  getListCollectionEntriesInfiniteQueryKey,
  getListMasterInventoryInfiniteQueryKey,
} from "@/generated/endpoints/inventory/inventory"
import { NETWORK_ERROR } from "@/lib/collections"
import { catalogCard, fakeBinder, stubBinderApi } from "@/test-binder-api"
import { stubGridLayout } from "@/test-grid-layout"

// CardTile links through TanStack Router's `Link`, which needs a router.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({
      children,
      params: _params,
      to,
      search: _search,
      ...props
    }: any) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
  }
})

// The Page's pending-review banner reads the signed-in user.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({
  useSession: () => ({
    user: { id: "user-1", name: null, email: null },
    isAuthenticated: true,
    isLoading: false,
  }),
}))

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    ),
  }
}

const pikachu = catalogCard("pikachu", "Pikachu", "025")
const bulbasaur = catalogCard("bulbasaur", "Bulbasaur", "001")

// The write, not the GET that refetches the Page after it.
function lastWrite(requests: { method: string }[]) {
  return requests.filter((request) => request.method !== "GET").at(-1)
}

// Opens the Slot's panel, then the catalog search, and places the named Card.
async function searchAndPlace(action: string, cardName: string) {
  fireEvent.click(screen.getByRole("button", { name: action }))
  fireEvent.change(screen.getByRole("searchbox", { name: "Card name" }), {
    target: { value: cardName.slice(0, 3) },
  })
  fireEvent.click(screen.getByRole("button", { name: "Search cards" }))
  fireEvent.click(
    await screen.findByRole("button", { name: `Place ${cardName}` })
  )
}

describe("binder Slot actions", () => {
  beforeEach(() => {
    stubGridLayout()
    // jsdom has no layout to scroll, so the panel's scroll into view is stubbed.
    Element.prototype.scrollIntoView = vi.fn()
  })

  it("places a searched Card in an empty Slot and refreshes the Page", async () => {
    const binder = fakeBinder({ catalog: [pikachu, bulbasaur] })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )
    await searchAndPlace("Place a Card", "Pikachu")

    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeTruthy()
    expect(
      screen.getByText("Cards and Unidentified: 1 of 80 slots")
    ).toBeTruthy()
    expect(lastWrite(binder.requests)).toMatchObject({
      method: "PUT",
      path: "/collections/col-1/binder/pages/1/slots/1/1",
      body: { cardId: "pikachu" },
    })
  })

  it("offers no Clear on an empty Slot", async () => {
    stubBinderApi(fakeBinder())
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )

    expect(screen.getByRole("button", { name: "Place a Card" })).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull()
    expect(
      screen.queryByRole("button", { name: "Mark Unidentified" })
    ).toBeNull()
  })

  it("offers Replace, Clear and Mark Unidentified on a filled Slot", async () => {
    stubBinderApi(
      fakeBinder({
        catalog: [pikachu],
        filled: { 1: { "1,1": { state: "card", card: pikachu } } },
      })
    )
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )

    expect(screen.getByRole("button", { name: "Replace card" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Clear" })).toBeTruthy()
    expect(
      screen.getByRole("button", { name: "Mark Unidentified" })
    ).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Place a Card" })).toBeNull()
  })

  it("clears a filled Slot back to empty", async () => {
    const binder = fakeBinder({
      catalog: [pikachu],
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Clear" }))

    expect(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    ).toBeTruthy()
    expect(lastWrite(binder.requests)).toMatchObject({
      method: "DELETE",
      path: "/collections/col-1/binder/pages/1/slots/1/1",
    })
    expect(
      screen.getByText("Cards and Unidentified: 0 of 80 slots")
    ).toBeTruthy()
  })

  it("replaces the Card in a filled Slot through the same search", async () => {
    const binder = fakeBinder({
      catalog: [pikachu, bulbasaur],
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    await searchAndPlace("Replace card", "Bulbasaur")

    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Bulbasaur, Base Set",
      })
    ).toBeTruthy()
    expect(lastWrite(binder.requests)).toMatchObject({
      method: "PUT",
      path: "/collections/col-1/binder/pages/1/slots/1/1",
      body: { cardId: "bulbasaur" },
    })
    expect(
      screen.getByText("Cards and Unidentified: 1 of 80 slots")
    ).toBeTruthy()
  })

  it("marks a filled Slot Unidentified", async () => {
    const binder = fakeBinder({
      catalog: [pikachu],
      filled: { 1: { "1,1": { state: "card", card: pikachu } } },
    })
    stubBinderApi(binder)
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Mark Unidentified" }))

    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Unidentified",
      })
    ).toBeTruthy()
    expect(lastWrite(binder.requests)).toMatchObject({
      method: "PUT",
      path: "/collections/col-1/binder/pages/1/slots/1/1/unidentified",
    })
    expect(screen.getByText("Unidentified: 1")).toBeTruthy()
  })

  it("gives a Card to an Unidentified Slot, and offers no Mark Unidentified there", async () => {
    stubBinderApi(
      fakeBinder({
        catalog: [pikachu],
        filled: { 1: { "1,1": { state: "unidentified" } } },
      })
    )
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Unidentified",
      })
    )
    expect(screen.getByRole("button", { name: "Clear" })).toBeTruthy()
    expect(
      screen.queryByRole("button", { name: "Mark Unidentified" })
    ).toBeNull()

    await searchAndPlace("Place a Card", "Pikachu")

    expect(
      await screen.findByRole("button", {
        name: "Row 1, column 1: Pikachu, Base Set",
      })
    ).toBeTruthy()
    expect(screen.getByText("Unidentified: 0")).toBeTruthy()
  })

  it.each([
    [400, "this Slot is outside the Layout"],
    [404, "no Card has that id"],
  ])(
    "shows a %i refused write inline and leaves the Slot as it was",
    async (status, detail) => {
      const binder = fakeBinder({
        catalog: [pikachu],
        writeError: { status, detail },
      })
      stubBinderApi(binder)
      renderWithClient(<BinderView collectionId="col-1" />)

      fireEvent.click(
        await screen.findByRole("button", { name: "Row 1, column 1: empty" })
      )
      fireEvent.click(screen.getByRole("button", { name: "Place a Card" }))
      fireEvent.change(screen.getByRole("searchbox", { name: "Card name" }), {
        target: { value: "Pik" },
      })
      fireEvent.click(screen.getByRole("button", { name: "Search cards" }))
      fireEvent.click(
        await screen.findByRole("button", { name: "Place Pikachu" })
      )

      expect((await screen.findByRole("alert")).textContent).toBe(detail)
      expect(
        screen.getByRole("button", { name: "Row 1, column 1: empty" })
      ).toBeTruthy()
    }
  )

  it("refreshes the Page, the entries, Master Inventory and the Collection after a placement", async () => {
    stubBinderApi(fakeBinder({ catalog: [pikachu] }))
    const { queryClient } = renderWithClient(
      <BinderView collectionId="col-1" />
    )
    const keys = [
      getListCollectionEntriesInfiniteQueryKey("col-1", { limit: 100 }),
      getListMasterInventoryInfiniteQueryKey({ limit: 100 }),
      getGetCollectionQueryKey("col-1"),
    ]
    for (const key of keys) queryClient.setQueryData(key, {})

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )
    await searchAndPlace("Place a Card", "Pikachu")

    await screen.findByRole("button", {
      name: "Row 1, column 1: Pikachu, Base Set",
    })
    for (const key of keys)
      expect(queryClient.getQueryState(key)?.isInvalidated).toBe(true)
  })

  it("moves focus into the panel when a Slot is tapped, and scrolls it into view", async () => {
    stubBinderApi(fakeBinder())
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )

    const panel = screen.getByRole("region", { name: "Slot actions" })
    expect(document.activeElement).toBe(panel)
    expect(panel.scrollIntoView).toHaveBeenCalledWith({ block: "nearest" })
  })

  it("puts focus in the name field when the catalog search opens", async () => {
    stubBinderApi(fakeBinder({ catalog: [pikachu] }))
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Place a Card" }))

    expect(document.activeElement).toBe(
      screen.getByRole("searchbox", { name: "Card name" })
    )
  })

  it("returns focus to the Slot it opened when the panel is closed", async () => {
    stubBinderApi(fakeBinder())
    renderWithClient(<BinderView collectionId="col-1" />)

    const tile = await screen.findByRole("button", {
      name: "Row 1, column 1: empty",
    })
    fireEvent.click(tile)
    fireEvent.click(screen.getByRole("button", { name: "Close" }))

    expect(screen.queryByRole("region", { name: "Slot actions" })).toBeNull()
    expect(document.activeElement).toBe(tile)
  })

  it("returns focus to the Slot after a write closes the panel", async () => {
    stubBinderApi(fakeBinder({ catalog: [pikachu] }))
    renderWithClient(<BinderView collectionId="col-1" />)

    const tile = await screen.findByRole("button", {
      name: "Row 1, column 1: empty",
    })
    fireEvent.click(tile)
    await searchAndPlace("Place a Card", "Pikachu")

    await screen.findByRole("button", {
      name: "Row 1, column 1: Pikachu, Base Set",
    })
    expect(screen.queryByRole("region", { name: "Slot actions" })).toBeNull()
    expect(document.activeElement).toBe(tile)
  })

  it("reports a network failure on a Slot write inline", async () => {
    stubBinderApi(fakeBinder({ catalog: [pikachu] }))
    renderWithClient(<BinderView collectionId="col-1" />)

    fireEvent.click(
      await screen.findByRole("button", { name: "Row 1, column 1: empty" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Place a Card" }))
    fireEvent.change(screen.getByRole("searchbox", { name: "Card name" }), {
      target: { value: "Pik" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Search cards" }))
    const place = await screen.findByRole("button", { name: "Place Pikachu" })
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch")
      })
    )
    fireEvent.click(place)

    expect(await screen.findByText(NETWORK_ERROR)).toBeTruthy()
  })
})
