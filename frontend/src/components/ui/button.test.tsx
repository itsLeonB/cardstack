import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
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

  it("defaults a rendered native <button> to type=button unless a type is given", () => {
    render(
      <>
        <Button render={<button />}>Plain</Button>
        <Button render={<button type="submit" />}>Element type</Button>
        <Button render={<button />} type="reset">
          Prop type
        </Button>
        <Button render={<Button />}>Nested</Button>
      </>
    )

    const type = (name: string) =>
      screen.getByRole("button", { name }).getAttribute("type")
    expect(type("Plain")).toBe("button")
    expect(type("Element type")).toBe("submit")
    expect(type("Prop type")).toBe("reset")
    expect(type("Nested")).toBe("button")
  })

  it("blocks activation of a disabled link and exposes aria-disabled", () => {
    const onClick = vi.fn()
    render(
      <>
        <Button disabled render={<a href="#a" onClick={onClick} />}>
          Off
        </Button>
        <Button render={<a href="#b" />}>On</Button>
      </>
    )

    const off = screen.getByRole("link", { name: "Off" })
    expect(off.getAttribute("aria-disabled")).toBe("true")
    expect(off.hasAttribute("disabled")).toBe(false)
    // fireEvent returns false when the click's default (navigation) was prevented
    expect(fireEvent.click(off)).toBe(false)

    const on = screen.getByRole("link", { name: "On" })
    expect(on.hasAttribute("aria-disabled")).toBe(false)
    expect(fireEvent.click(on)).toBe(true)
  })
})
