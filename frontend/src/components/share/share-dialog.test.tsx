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

  it("disables the switch while a change is in flight", async () => {
    renderDialog({ shared: false, pending: true })
    fireEvent.click(screen.getByRole("button", { name: "Share" }))

    const toggle = await screen.findByRole("switch", {
      name: "Anyone with the link can view",
    })
    expect(
      toggle.hasAttribute("disabled") ||
        toggle.getAttribute("aria-disabled") === "true"
    ).toBe(true)
  })

  it("keeps sharing when Keep sharing is chosen, without calling onUnshare", async () => {
    const { onUnshare } = renderDialog({ shared: true })
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))
    fireEvent.click(
      await screen.findByRole("switch", {
        name: "Anyone with the link can view",
      })
    )

    fireEvent.click(screen.getByRole("button", { name: "Keep sharing" }))

    expect(screen.queryByText(/Existing links will stop working/)).toBeNull()
    expect(screen.getByRole("textbox", { name: "Share link" })).toBeTruthy()
    expect(onUnshare).not.toHaveBeenCalled()
  })

  it("shows the error inline as an alert", async () => {
    renderDialog({ shared: false, error: "Could not share this Collection." })
    fireEvent.click(screen.getByRole("button", { name: "Share" }))

    const alert = await screen.findByRole("alert")
    expect(alert.textContent).toBe("Could not share this Collection.")
  })

  it("says so when the link cannot be copied", async () => {
    vi.stubGlobal("navigator", {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
    })
    renderDialog({ shared: true })
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))

    fireEvent.click(await screen.findByRole("button", { name: "Copy" }))

    expect(await screen.findByText(/Could not copy the link/)).toBeTruthy()
    expect(screen.queryByText("Copied")).toBeNull()
  })

  it("clears the message at once on a repeat copy, so it is announced again", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    renderDialog({ shared: true })
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))
    fireEvent.click(await screen.findByRole("button", { name: "Copy" }))
    await screen.findByText("Copied")

    fireEvent.click(screen.getByRole("button", { name: "Copy" }))
    const live = document.querySelector('[aria-live="polite"]')
    expect(live?.textContent).toBe("")

    await screen.findByText("Copied")
    expect(writeText).toHaveBeenCalledTimes(2)
  })

  it("drops the Copied message when the link changes on a re-share", async () => {
    vi.stubGlobal("navigator", {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
    const view = render(
      <ShareDialog shared link={LINK} onShare={vi.fn()} onUnshare={vi.fn()} />
    )
    fireEvent.click(screen.getByRole("button", { name: "Shared" }))
    fireEvent.click(await screen.findByRole("button", { name: "Copy" }))
    await screen.findByText("Copied")

    view.rerender(
      <ShareDialog
        shared
        link={`${LINK}-next`}
        onShare={vi.fn()}
        onUnshare={vi.fn()}
      />
    )

    await vi.waitFor(() => expect(screen.queryByText("Copied")).toBeNull())
  })
})
