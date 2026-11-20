import { afterEach, describe, expect, it } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { TradeShare } from "@/components/share/trade-share"

afterEach(cleanup)

describe("TradeShare", () => {
  it("opens a named link dialog with delivery actions and no publication control", () => {
    const link = "https://cardstack.test/trades/trade-1"
    render(<TradeShare link={link} />)

    fireEvent.click(screen.getByRole("button", { name: "Invite" }))

    expect(
      screen.getByRole("dialog", { name: "Share Trade link" })
    ).toBeTruthy()
    expect(screen.getByRole("textbox", { name: "Trade link" })).toHaveProperty(
      "value",
      link
    )
    expect(screen.getByRole("button", { name: "Copy" })).toBeTruthy()
    expect(screen.queryByRole("switch")).toBeNull()
    expect(screen.queryByText(/anyone with the link/i)).toBeNull()
  })
})
