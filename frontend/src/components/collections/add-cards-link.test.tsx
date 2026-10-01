import { afterEach, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { AddCardsLink } from "./add-cards-link"

// `Link` needs a router in the tree; expose its target and search as attributes instead.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({ children, to, search }: { children: string; to: string; search: unknown }) => (
      <a href={to} data-search={JSON.stringify(search)}>
        {children}
      </a>
    ),
  }
})

afterEach(cleanup)

it("links to the catalog with the Collection preselected", () => {
  render(<AddCardsLink collectionId="col-1" />)
  const link = screen.getByRole("link", { name: "Add Cards" })
  expect(link.getAttribute("href")).toBe("/catalog/search")
  expect(link.getAttribute("data-search")).toBe('{"collectionId":"col-1"}')
})
