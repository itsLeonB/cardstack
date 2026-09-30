import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { CollectionForm } from "./collection-form"

afterEach(() => cleanup())

function fillAndSubmit({
  title,
  description,
  maxCardCount,
}: {
  title?: string
  description?: string
  maxCardCount?: string
}) {
  if (title !== undefined) {
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: title },
    })
  }
  if (description !== undefined) {
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: description },
    })
  }
  if (maxCardCount !== undefined) {
    fireEvent.change(screen.getByLabelText("Max card count"), {
      target: { value: maxCardCount },
    })
  }
  fireEvent.click(screen.getByRole("button", { name: "Save" }))
}

describe("CollectionForm", () => {
  it("submits the trimmed title/description and a parsed maxCardCount", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({
      title: "  Vintage binder  ",
      description: "  Base Set only  ",
      maxCardCount: "50",
    })

    expect(onSubmit).toHaveBeenCalledWith({
      title: "Vintage binder",
      description: "Base Set only",
      maxCardCount: 50,
    })
  })

  it("sends maxCardCount 0 and omits description when left blank", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "Vintage binder" })

    expect(onSubmit).toHaveBeenCalledWith({
      title: "Vintage binder",
      description: undefined,
      maxCardCount: 0,
    })
  })

  it("rejects a blank title without calling onSubmit", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "   " })

    expect(onSubmit).not.toHaveBeenCalled()
    screen.getByText("Title is required.")
  })

  it("accepts an explicit 0 as no limit", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "Vintage binder", maxCardCount: "0" })

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ maxCardCount: 0 })
    )
  })

  it("rejects a negative or non-integer maxCardCount without calling onSubmit", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "Vintage binder", maxCardCount: "-1" })

    expect(onSubmit).not.toHaveBeenCalled()
    screen.getByText("Max card count must be a non-negative whole number.")
  })

  it("rejects a maxCardCount above the backend's int32 cap", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "Vintage binder", maxCardCount: "2147483648" })

    expect(onSubmit).not.toHaveBeenCalled()
    screen.getByText("Max card count can't exceed 2,147,483,647.")
  })

  it("accepts a maxCardCount at the int32 cap", () => {
    const onSubmit = vi.fn()
    render(
      <CollectionForm
        submitLabel="Save"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={onSubmit}
      />
    )

    fillAndSubmit({ title: "Vintage binder", maxCardCount: "2147483647" })

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ maxCardCount: 2147483647 })
    )
  })

  it("pre-fills fields from initialValues (edit mode)", () => {
    render(
      <CollectionForm
        initialValues={{
          title: "Vintage binder",
          description: "Base Set only",
          maxCardCount: "50",
        }}
        submitLabel="Save changes"
        pendingLabel="Saving..."
        isPending={false}
        onSubmit={vi.fn()}
      />
    )

    // SAFETY: these labeled fields are all rendered as <input> elements.
    expect((screen.getByLabelText("Title") as HTMLInputElement).value).toBe(
      "Vintage binder"
    )
    // SAFETY: these labeled fields are all rendered as <input> elements.
    expect(
      (screen.getByLabelText("Description") as HTMLInputElement).value
    ).toBe("Base Set only")
    // SAFETY: these labeled fields are all rendered as <input> elements.
    expect(
      (screen.getByLabelText("Max card count") as HTMLInputElement).value
    ).toBe("50")
  })
})
