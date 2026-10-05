import { afterEach, expect, it } from "vitest"
import { cleanup, fireEvent, render } from "@testing-library/react"
import { SeriesHeading } from "./series-heading"

afterEach(cleanup)

it("tries a new logo address again after the previous one failed", () => {
  const heading = (imageUrl: string) => (
    <SeriesHeading id="s" name="Scarlet & Violet" imageUrl={imageUrl} />
  )
  const { container, rerender } = render(heading("https://img.test/a.webp"))

  fireEvent.error(container.querySelector("img")!)
  expect(container.querySelector("img")).toBeNull()

  rerender(heading("https://img.test/b.webp"))
  expect(container.querySelector("img")?.getAttribute("src")).toBe(
    "https://img.test/b.webp"
  )
})
