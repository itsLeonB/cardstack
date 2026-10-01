import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { MasterInventory } from "./master-inventory"
import { useListMasterInventory } from "@/generated/endpoints/inventory/inventory"
import type { CardSummary } from "@/generated/models"

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/inventory/inventory", () => ({ useListMasterInventory: vi.fn() }))
// CardTile links via TanStack Router's `Link`, which needs a router in the tree.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params, to, ...props }: any) => (
      <a href={to.replace("$expansionSetId", params?.expansionSetId).replace("$localId", params?.localId)} {...props}>
        {children}
      </a>
    ),
  }
})

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

interface InventoryState {
  isPending: boolean
  isError: boolean
  data?: { status: number; data: { data?: unknown[]; meta?: unknown; detail?: string } }
}

function setup(state: InventoryState, page = 1) {
  // SAFETY: tests supply only the fields MasterInventory reads from this hook.
  vi.mocked(useListMasterInventory).mockReturnValue(state as any)
  const onPageChange = vi.fn()
  render(<MasterInventory page={page} onPageChange={onPageChange} />)
  return onPageChange
}
const ok = (data: unknown[], total = data.length) => ({
  isPending: false,
  isError: false,
  data: { status: 200, data: { data, meta: { total, page: 1, limit: 24 } } },
})

describe("MasterInventory", () => {
  it("shows each owned Card with its total quantity, linking to the card detail page", () => {
    setup(ok([{ card, quantity: 5 }]))
    expect(screen.getByText("×5")).toBeTruthy()
    expect(screen.getByRole("link", { name: "Pikachu V" }).getAttribute("href")).toBe("/catalog/cards/set-1/048")
    expect(screen.queryByLabelText("Quantity of Pikachu V")).toBeNull()
  })

  it("shows an empty state when the user owns nothing", () => {
    setup(ok([]))
    expect(screen.getByText(/don.t own any Cards yet/)).toBeTruthy()
  })

  it("always refetches and never keeps the list cached, so Collection changes show up", () => {
    setup(ok([]), 2)
    const [params, options] = vi.mocked(useListMasterInventory).mock.lastCall ?? []
    expect(params).toEqual({ page: 2 })
    expect(options?.query).toMatchObject({ gcTime: 0, refetchOnMount: "always" })
  })

  it("shows the API error detail for a non-200 response", () => {
    setup({ isPending: false, isError: false, data: { status: 500, data: { detail: "boom" } } })
    expect(screen.getByRole("alert").textContent).toBe("boom")
  })

  it("pages through the inventory", () => {
    const onPageChange = setup(ok([{ card, quantity: 1 }], 50))
    fireEvent.click(screen.getByRole("button", { name: /Next/ }))
    expect(onPageChange).toHaveBeenCalledWith(2)
  })
})
