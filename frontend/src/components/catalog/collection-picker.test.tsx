import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, renderHook, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { CollectionPicker, useCatalogCollection } from "./collection-picker"
import { useListCollections } from "@/generated/endpoints/collections/collections"
import { useSession } from "@/lib/session"

// Isolates the UI from the network and session probe.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/collections/collections", () => ({ useListCollections: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/session", () => ({ useSession: vi.fn() }))
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    // SAFETY: test stand-in for Link; props are plain anchor attributes.
    Link: ({ children, search: _search, to: _to, ...props }: any) => <a {...props}>{children}</a>,
  }
})

afterEach(() => cleanup())

const collections = [{ id: "c1", title: "Binder" }]

function setup({ authed, list }: { authed: boolean; list?: unknown }) {
  // SAFETY: partial mocks; the hook reads only these fields.
  vi.mocked(useSession).mockReturnValue({ isAuthenticated: authed, isLoading: false } as any)
  // SAFETY: partial mock; only data is read.
  vi.mocked(useListCollections).mockReturnValue({
    data: list === undefined ? undefined : { status: 200, data: { data: list } },
  } as any)
}

describe("CollectionPicker", () => {
  it("is disabled with a login prompt for guests", () => {
    render(<CollectionPicker isAuthenticated={false} collections={[]} value={undefined} onChange={vi.fn()} />)
    // SAFETY: the labelled control is a <select>.
    expect((screen.getByLabelText("Add to collection") as HTMLSelectElement).disabled).toBe(true)
    screen.getByText("Log in to add cards to a Collection")
  })

  it("lists Collections and reports the selection when authenticated", () => {
    const onChange = vi.fn()
    render(<CollectionPicker isAuthenticated collections={collections} value={undefined} onChange={onChange} />)
    fireEvent.change(screen.getByLabelText("Add to collection"), { target: { value: "c1" } })
    expect(onChange).toHaveBeenCalledWith("c1")
    expect(screen.queryByText(/Log in/)).toBeNull()
  })
})

describe("useCatalogCollection", () => {
  it("keeps a valid selection", () => {
    setup({ authed: true, list: collections })
    const onSelect = vi.fn()
    const { result } = renderHook(() => useCatalogCollection("c1", onSelect))
    expect(result.current.selected).toBe("c1")
    expect(onSelect).not.toHaveBeenCalled()
  })

  it("clears a Collection that no longer exists", () => {
    setup({ authed: true, list: collections })
    const onSelect = vi.fn()
    renderHook(() => useCatalogCollection("gone", onSelect))
    expect(onSelect).toHaveBeenCalledWith(undefined)
  })

  it("clears the selection for a guest", () => {
    setup({ authed: false })
    const onSelect = vi.fn()
    const { result } = renderHook(() => useCatalogCollection("c1", onSelect))
    expect(result.current.selected).toBeUndefined()
    expect(onSelect).toHaveBeenCalledWith(undefined)
  })

  it("waits for the list before judging the selection", () => {
    setup({ authed: true })
    const onSelect = vi.fn()
    renderHook(() => useCatalogCollection("c1", onSelect))
    expect(onSelect).not.toHaveBeenCalled()
  })
})
