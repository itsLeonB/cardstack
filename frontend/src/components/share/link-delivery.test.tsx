import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { LinkDelivery } from "@/components/share/link-delivery"

const LINK = "https://cardstack.test/trades/trade-1"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("LinkDelivery", () => {
  it("copies the current selectable link and announces repeated success", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    render(<LinkDelivery link={LINK} label="Trade link" />)

    expect(screen.getByRole("textbox", { name: "Trade link" })).toHaveProperty(
      "value",
      LINK
    )
    const copy = screen.getByRole("button", { name: "Copy" })
    fireEvent.click(copy)
    expect(await screen.findByText("Copied")).toBeTruthy()
    fireEvent.click(copy)
    expect(await screen.findByText("Copied")).toBeTruthy()
    expect(writeText).toHaveBeenCalledTimes(2)
  })

  it.each([
    ["missing", undefined],
    ["rejected", vi.fn().mockRejectedValue(new Error("denied"))],
  ])(
    "keeps manual copy guidance when clipboard is %s",
    async (_case, writeText) => {
      vi.stubGlobal("navigator", {
        clipboard: writeText ? { writeText } : undefined,
      })
      render(<LinkDelivery link={LINK} />)
      fireEvent.click(screen.getByRole("button", { name: "Copy" }))

      expect(
        await screen.findByText(/Select it and copy it by hand/)
      ).toBeTruthy()
      expect(
        screen.getByRole("textbox", { name: "Share link" })
      ).toHaveProperty("value", LINK)
    }
  )

  it("shares from a user action and treats cancellation as quiet", async () => {
    const share = vi
      .fn()
      .mockRejectedValue(new DOMException("cancelled", "AbortError"))
    vi.stubGlobal("navigator", { share })
    render(<LinkDelivery link={LINK} />)

    fireEvent.click(screen.getByRole("button", { name: "Share…" }))
    expect(share).toHaveBeenCalledWith({ url: LINK })
    expect(screen.queryByRole("status")?.textContent).toBe("")
  })

  it("shows native-share failures without removing Copy", async () => {
    vi.stubGlobal("navigator", {
      share: vi.fn().mockRejectedValue(new Error("failed")),
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
    render(<LinkDelivery link={LINK} />)

    fireEvent.click(screen.getByRole("button", { name: "Share…" }))
    expect(await screen.findByText(/Could not share the link/)).toBeTruthy()
    expect(screen.getByRole("button", { name: "Copy" })).toBeTruthy()
  })

  it("hides native sharing when unsupported", () => {
    vi.stubGlobal("navigator", { clipboard: { writeText: vi.fn() } })
    render(<LinkDelivery link={LINK} />)
    expect(screen.queryByRole("button", { name: "Share…" })).toBeNull()
    expect(screen.getByRole("button", { name: "Copy" })).toBeTruthy()
  })
})
