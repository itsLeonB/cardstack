import { afterEach, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import type * as TanStackRouter from "@tanstack/react-router"
import { ScanLink } from "./scan-link"

// `Link` needs a router in the tree; expose its target as attributes instead.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()
  return {
    ...actual,
    Link: ({
      children,
      to,
      params,
    }: {
      children: string
      to: string
      params: { collectionId: string }
    }) => (
      <a href={to.replace("$collectionId", params.collectionId)}>{children}</a>
    ),
  }
})

afterEach(() => {
  cleanup()
  vi.unstubAllEnvs()
})

it("links to the Collection's scan screen when the flag is on", () => {
  vi.stubEnv("VITE_SCAN_ENABLED", "true")
  render(<ScanLink collectionId="col-1" />)
  expect(
    screen.getByRole("link", { name: "Scan cards" }).getAttribute("href")
  ).toBe("/collections/col-1/scan")
})

it.each<string | undefined>([undefined, "", "false", "1"])(
  "renders nothing when the flag is %j",
  (value) => {
    vi.stubEnv("VITE_SCAN_ENABLED", value)
    render(<ScanLink collectionId="col-1" />)
    expect(screen.queryByRole("link")).toBeNull()
  }
)
