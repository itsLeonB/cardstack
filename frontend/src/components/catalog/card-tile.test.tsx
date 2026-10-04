import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { CardTile } from "./card-tile"
import type { CardSummary } from "@/generated/models"

const HOST = "https://img.example.com"

beforeEach(() => vi.stubEnv("VITE_IMAGE_HOST", HOST))
afterEach(() => {
  cleanup()
  vi.unstubAllEnvs()
})

// `Link` needs a <RouterProvider>; swap it for a plain anchor (same approach as expansion-set-tile.test.tsx).
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

const card: CardSummary = {
  id: "card-1",
  localId: "001",
  name: "Pikachu",
  category: "Pokemon",
  illustrator: "Ken Sugimori",
  imageUrl: `${HOST}/cards/card-1`,
  rarity: { id: "r1", code: "C", name: "Common" },
  expansionSet: { id: "set-1", code: "M2", name: "Inferno X", imageUrl: "" },
  tags: [],
}

const sized = (width: number) =>
  `${HOST}/cdn-cgi/image/width=${width},format=auto,onerror=redirect/cards/card-1`

describe("CardTile image", () => {
  it("requests 240 and 480 wide sources and declares its dimensions", () => {
    const { container } = render(<CardTile card={card} />)
    const img = container.querySelector("img")!

    expect(img.getAttribute("src")).toBe(sized(240))
    expect(img.getAttribute("srcset")).toBe(
      `${sized(240)} 1x, ${sized(480)} 2x`
    )
    expect(img.getAttribute("width")).toBe("240")
    expect(img.getAttribute("height")).toBe("336")
  })

  it("requests the 720 wide source in the detail variant", () => {
    const { container } = render(<CardTile card={card} imageVariant="detail" />)
    const img = container.querySelector("img")!

    expect(img.getAttribute("src")).toBe(sized(720))
    expect(img.getAttribute("srcset")).toBeNull()
    expect(img.getAttribute("width")).toBe("720")
    expect(img.getAttribute("height")).toBe("1008")
  })

  it("shows the name placeholder when the address is empty", () => {
    const { container } = render(<CardTile card={{ ...card, imageUrl: "" }} />)

    expect(container.querySelector("img")).toBeNull()
    expect(screen.getAllByText("Pikachu").length).toBeGreaterThan(1)
  })

  it("leaves an off-host address unchanged", () => {
    const offHost = "https://other.example.net/cards/card-1.png"
    const { container } = render(
      <CardTile card={{ ...card, imageUrl: offHost }} />
    )
    const img = container.querySelector("img")!

    expect(img.getAttribute("src")).toBe(offHost)
    expect(img.getAttribute("srcset")).toBeNull()
  })

  it("falls back to the name placeholder when the image fails to load", () => {
    const { container } = render(<CardTile card={card} />)

    fireEvent.error(container.querySelector("img")!)

    expect(container.querySelector("img")).toBeNull()
    expect(screen.getAllByText("Pikachu").length).toBeGreaterThan(1)
  })
})
