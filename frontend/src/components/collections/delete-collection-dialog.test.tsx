import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { DeleteCollectionDialog } from "./delete-collection-dialog"
import { useDeleteCollectionMutation } from "@/lib/collections"

// Isolates the dialog's confirmation gating from the real mutation/network
// stack; see delete-collection-dialog.tsx's seam and lib/collections.test.tsx
// for the mutation wrapper's own invalidation tests.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/lib/collections", () => ({
  useDeleteCollectionMutation: vi.fn(),
}))

const mockUseDeleteCollectionMutation = vi.mocked(useDeleteCollectionMutation)

afterEach(() => cleanup())

describe("DeleteCollectionDialog", () => {
  const mutate = vi.fn()

  beforeEach(() => {
    mutate.mockReset()
    // SAFETY: partial mock; only mutate/isPending are read by the component.
    mockUseDeleteCollectionMutation.mockReturnValue({
      mutate,
      isPending: false,
    } as any)
  })

  it("does not fire the DELETE mutation when only the trigger is clicked", () => {
    render(
      <DeleteCollectionDialog
        collectionId="col-1"
        collectionTitle="Vintage binder"
      />
    )

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Vintage binder" })
    )

    screen.getByRole("alertdialog")
    expect(mutate).not.toHaveBeenCalled()
  })

  it("fires the DELETE mutation only after the confirm action is clicked", () => {
    render(
      <DeleteCollectionDialog
        collectionId="col-1"
        collectionTitle="Vintage binder"
      />
    )

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Vintage binder" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Delete" }))

    expect(mutate).toHaveBeenCalledTimes(1)
    expect(mutate).toHaveBeenCalledWith(
      { id: "col-1" },
      expect.objectContaining({ onSuccess: expect.any(Function) })
    )
  })

  it("does not fire the DELETE mutation when cancel is clicked", () => {
    render(
      <DeleteCollectionDialog
        collectionId="col-1"
        collectionTitle="Vintage binder"
      />
    )

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Vintage binder" })
    )
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))

    expect(mutate).not.toHaveBeenCalled()
  })
})
