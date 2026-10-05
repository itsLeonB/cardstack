import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { ExpansionSetTile } from "./expansion-set-tile"
import type { ExpansionSetSummary } from "@/generated/models"

afterEach(() => {
  cleanup()
  vi.unstubAllEnvs()
})

// `Link` needs a <RouterProvider>; swap it for a plain anchor (same approach as card-results.test.tsx).
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, params: _params, to: _to, ...props }: any) => (
      <a {...props}>{children}</a>
    ),
  }
})

const expansionSet: ExpansionSetSummary = {
  id: "set-1",
  code: "M2",
  name: "Inferno X",
  releaseDate: "2025-01-01",
  imageUrl: "https://example.com/m2.png",
}

describe("ExpansionSetTile", () => {
  it("renders the cover image when imageUrl is set", () => {
    const { container } = render(
      <ExpansionSetTile expansionSet={expansionSet} />
    )

    expect(container.querySelector("img")?.getAttribute("src")).toBe(
      expansionSet.imageUrl
    )
  })

  it("loads the hosted cover as-is, even from the image host, and keeps the 64px slot", () => {
    vi.stubEnv("VITE_IMAGE_HOST", "https://img.example.com")
    const imageUrl =
      "https://img.example.com/expansion-sets/set-1.ab12cd34.webp"
    const { container } = render(
      <ExpansionSetTile expansionSet={{ ...expansionSet, imageUrl }} />
    )
    const img = container.querySelector("img")!

    expect(img.getAttribute("src")).toBe(imageUrl)
    expect(img.getAttribute("width")).toBe("64")
    expect(img.getAttribute("height")).toBe("64")
  })

  it("renders a text-only tile when imageUrl is empty", () => {
    const { container } = render(
      <ExpansionSetTile expansionSet={{ ...expansionSet, imageUrl: "" }} />
    )

    expect(container.querySelector("img")).toBeNull()
    expect(
      container.querySelector("[data-slot=card]")?.className
    ).not.toContain("flex-row")
    expect(screen.getByText("Inferno X")).toBeTruthy()
  })

  it("falls back to the text-only tile when the image fails to load", () => {
    const { container } = render(
      <ExpansionSetTile expansionSet={expansionSet} />
    )

    fireEvent.error(container.querySelector("img")!)

    expect(container.querySelector("img")).toBeNull()
    expect(screen.getByText("Inferno X")).toBeTruthy()
  })
})
