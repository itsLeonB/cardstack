import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import { Button } from "./button"

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe("Button", () => {
  it("renders a link as a plain <a> without button semantics or a Base UI warning", () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => {})
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {})

    render(
      <Button variant="outline" render={<a href="/catalog" />}>
        Browse
      </Button>
    )

    const link = screen.getByRole("link", { name: "Browse" })
    expect(link.tagName).toBe("A")
    expect(link.hasAttribute("role")).toBe(false)
    expect(link.getAttribute("href")).toBe("/catalog")
    expect(link.dataset.slot).toBe("button")
    expect(link.className).toContain("border-border")
    expect(error).not.toHaveBeenCalled()
    expect(warn).not.toHaveBeenCalled()
  })

  it("still renders a native <button> by default", () => {
    render(<Button>Save</Button>)

    const button = screen.getByRole("button", { name: "Save" })
    expect(button.tagName).toBe("BUTTON")
    expect(button.dataset.slot).toBe("button")
  })
})
