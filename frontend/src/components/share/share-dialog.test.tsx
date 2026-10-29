import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { ShareDialog } from "@/components/share/share-dialog"

const LINK = "https://cardstack.test/s/profile-ash/holiday-binder-3f9a2"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function renderDialog(props: Partial<Parameters<typeof ShareDialog>[0]> = {}) {
  const callbacks = { onShare: vi.fn(), onUnshare: vi.fn() }
  render(<ShareDialog shared={false} link={LINK} {...callbacks} {...props} />)
  return callbacks
}

describe("ShareDialog", () => {
  it("names the sharing state on its trigger", () => {
    renderDialog({ shared: false })
    expect(screen.getByRole("button", { name: "Share" })).toBeTruthy()
    cleanup()

    renderDialog({ shared: true })
    expect(screen.getByRole("button", { name: "Shared" })).toBeTruthy()
  })

  it("warns about the owner's full name above the switch before sharing is first turned on", async () => {
    const { onShare } = renderDialog({ shared: false })
    fireEvent.click(screen.getByRole("button", { name: "Share" }))

    const warning = await screen.findByText(/full name/i)
    const toggle = screen.getByRole("switch", {
      name: "Anyone with the link can view",
    })
    expect(
      warning.compareDocumentPosition(toggle) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()

    fireEvent.click(toggle)
    expect(onShare).toHaveBeenCalledOnce()
  })

  it("shows the read-only link and announces Copied after Copy", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    renderDialog({ shared: true })
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))

    const field = await screen.findByRole("textbox", { name: "Share link" })
    expect(screen.getByDisplayValue(LINK)).toBe(field)
    expect(field.hasAttribute("readonly")).toBe(true)

    fireEvent.click(screen.getByRole("button", { name: "Copy" }))

    expect(writeText).toHaveBeenCalledWith(LINK)
    const status = await screen.findByText("Copied")
    expect(status.getAttribute("aria-live")).toBe("polite")
  })

  it("asks for confirmation before turning sharing off, and only then ends it", async () => {
    const { onUnshare } = renderDialog({ shared: true })
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))

    fireEvent.click(
      await screen.findByRole("switch", {
        name: "Anyone with the link can view",
      })
    )
    expect(
      await screen.findByText(/Existing links will stop working/)
    ).toBeTruthy()
    expect(onUnshare).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("button", { name: "Stop sharing" }))
    expect(onUnshare).toHaveBeenCalledOnce()
  })
})
