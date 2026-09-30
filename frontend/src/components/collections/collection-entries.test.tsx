import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CollectionEntries } from "./collection-entries"
import {
  useAddCollectionEntry,
  useListCollectionEntries,
  useRemoveCollectionEntry,
  useUpdateCollectionEntry,
} from "@/generated/endpoints/inventory/inventory"
import { useSearchCatalogCards } from "@/generated/endpoints/catalog/catalog"
import type { CardSummary } from "@/generated/models"

// Isolates the UI from the network; the generated hooks' wire behaviour is
// orval's job.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({
  getListCollectionEntriesQueryKey: (id: string) => ["entries", id],
  useListCollectionEntries: vi.fn(),
  useAddCollectionEntry: vi.fn(),
  useUpdateCollectionEntry: vi.fn(),
  useRemoveCollectionEntry: vi.fn(),
}))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/catalog/catalog", () => ({
  useSearchCatalogCards: vi.fn(),
}))

afterEach(() => cleanup())

const card: CardSummary = {
  id: "card-1",
  name: "Pikachu V",
  localId: "048",
  category: "Pokémon",
  tags: [],
  illustrator: "someone",
  imageUrl: "",
  expansionSet: { id: "set-1", code: "SCE", name: "Starter" },
  rarity: { id: "r-1", code: "RR", name: "Double Rare" },
}

const update = vi.fn()
const remove = vi.fn()
const add = vi.fn()

function setList(items: { card: CardSummary; quantity: number }[]) {
  // SAFETY: partial mock; the component reads only status/data and isPending/isError.
  vi.mocked(useListCollectionEntries).mockReturnValue({
    isPending: false,
    isError: false,
    data: { status: 200, data: { data: items }, headers: new Headers() },
  } as any)
}

function renderEntries() {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <CollectionEntries collectionId="col-1" />
    </QueryClientProvider>
  )
}

describe("CollectionEntries", () => {
  beforeEach(() => {
    for (const fn of [update, remove, add]) fn.mockReset()
    // SAFETY: partial mock; only mutate/isPending are read.
    vi.mocked(useUpdateCollectionEntry).mockReturnValue({
      mutate: update,
      isPending: false,
    } as any)
    // SAFETY: partial mock; only mutate/isPending are read.
    vi.mocked(useRemoveCollectionEntry).mockReturnValue({
      mutate: remove,
      isPending: false,
    } as any)
    // SAFETY: partial mock; only mutate/isPending are read.
    vi.mocked(useAddCollectionEntry).mockReturnValue({
      mutate: add,
      isPending: false,
    } as any)
    // SAFETY: partial mock; only status/data/isError are read.
    vi.mocked(useSearchCatalogCards).mockReturnValue({
      isError: false,
      data: { status: 200, data: { data: [card] }, headers: new Headers() },
    } as any)
  })

  it("shows an empty state when the Collection has no Cards", () => {
    setList([])
    renderEntries()
    screen.getByText(/has no Cards yet/)
  })

  it("lists each Card with its quantity", () => {
    setList([{ card, quantity: 3 }])
    renderEntries()
    screen.getByText("Pikachu V")
    // SAFETY: the labelled control is an <input>.
    const input = screen.getByLabelText("Quantity of Pikachu V") as HTMLInputElement
    expect(input.value).toBe("3")
  })

  it("shows a capacity rejection and restores the displayed quantity", () => {
    setList([{ card, quantity: 3 }])
    update.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 422, data: { detail: "Capacity exceeded" } })
    )
    renderEntries()

    // SAFETY: the labelled control is an <input>.
    const input = screen.getByLabelText("Quantity of Pikachu V") as HTMLInputElement
    fireEvent.change(input, { target: { value: "50" } })
    fireEvent.click(screen.getByRole("button", { name: "Save quantity of Pikachu V" }))

    expect(update).toHaveBeenCalledWith(
      { id: "col-1", cardId: "card-1", data: { quantity: 50 } },
      expect.anything()
    )
    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
    expect(input.value).toBe("3")
  })

  it("removes a Card without confirmation", () => {
    setList([{ card, quantity: 3 }])
    renderEntries()
    fireEvent.click(screen.getByRole("button", { name: "Remove Pikachu V" }))
    expect(remove).toHaveBeenCalledWith({ id: "col-1", cardId: "card-1" }, expect.anything())
  })

  it("adds a searched Card with a quantity and surfaces a duplicate (409)", () => {
    setList([])
    add.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 409, data: { detail: "Card already in collection" } })
    )
    renderEntries()

    fireEvent.change(screen.getByLabelText("Search Cards to add"), { target: { value: "pika" } })
    fireEvent.click(screen.getByRole("button", { name: "Search" }))
    fireEvent.change(screen.getByLabelText("Quantity to add of Pikachu V"), {
      target: { value: "2" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Add Pikachu V" }))

    expect(add).toHaveBeenCalledWith(
      { id: "col-1", data: { cardId: "card-1", quantity: 2 } },
      expect.anything()
    )
    expect(screen.getByRole("alert").textContent).toBe("Card already in collection")
  })

  function saveWith(status: number, detail = "boom") {
    update.mockImplementation((_vars, options) => options.onSuccess({ status, data: { detail } }))
    fireEvent.change(screen.getByLabelText("Quantity of Pikachu V"), { target: { value: "5" } })
    fireEvent.click(screen.getByRole("button", { name: "Save quantity of Pikachu V" }))
  }

  it("refetches the list when update or remove hits a 404", () => {
    setList([{ card, quantity: 3 }])
    const client = new QueryClient()
    const spy = vi.spyOn(client, "invalidateQueries")
    render(
      <QueryClientProvider client={client}>
        <CollectionEntries collectionId="col-1" />
      </QueryClientProvider>
    )

    saveWith(404, "Card not in collection")
    expect(screen.getByRole("alert").textContent).toBe("Card not in collection")
    expect(spy).toHaveBeenCalledTimes(1)

    remove.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 404, data: { detail: "gone" } })
    )
    fireEvent.click(screen.getByRole("button", { name: "Remove Pikachu V" }))
    expect(screen.getByRole("alert").textContent).toBe("gone")
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it("shows the generic message on a network error", () => {
    setList([{ card, quantity: 3 }])
    update.mockImplementation((_vars, options) => options.onError(new Error("x")))
    renderEntries()
    fireEvent.change(screen.getByLabelText("Quantity of Pikachu V"), { target: { value: "5" } })
    fireEvent.click(screen.getByRole("button", { name: "Save quantity of Pikachu V" }))
    expect(screen.getByRole("alert").textContent).toBe(
      "Could not reach the server. Please try again."
    )
  })

  it("shows the message when adding is rejected for capacity (422)", () => {
    setList([])
    add.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 422, data: { detail: "Capacity exceeded" } })
    )
    renderEntries()
    fireEvent.change(screen.getByLabelText("Search Cards to add"), { target: { value: "pika" } })
    fireEvent.click(screen.getByRole("button", { name: "Search" }))
    fireEvent.click(screen.getByRole("button", { name: "Add Pikachu V" }))
    expect(screen.getByRole("alert").textContent).toBe("Capacity exceeded")
  })

  it("clears the search results after a successful add", () => {
    setList([])
    add.mockImplementation((_vars, options) =>
      options.onSuccess({ status: 201, data: { data: { cardId: "card-1", quantity: 1 } } })
    )
    renderEntries()
    fireEvent.change(screen.getByLabelText("Search Cards to add"), { target: { value: "pika" } })
    fireEvent.click(screen.getByRole("button", { name: "Search" }))
    fireEvent.click(screen.getByRole("button", { name: "Add Pikachu V" }))
    expect(screen.queryByRole("button", { name: "Add Pikachu V" })).toBeNull()
  })
})
