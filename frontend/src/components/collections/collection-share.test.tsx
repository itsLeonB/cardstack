import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CollectionShare } from "@/components/collections/collection-share"

// The header's Share button against a faked network: the Collection's slug
// is the server's state, so a share or unshare only shows once the Collection
// is read back. The generated client, the query cache and the mutations are
// the app's own.

const COLLECTION = "/collections/col-1"
const SLUG = "holiday-binder-3f9a2"

// The JSON bodies the API answers with: an envelope, or an error detail.
interface ApiBody {
  data?: unknown
  detail?: string
}

interface Server {
  slug: string | null
  profileStatus: number
}

// Each request is answered by path and method, and the calls made are kept.
function stubServer(server: Server) {
  const calls: string[] = []
  const reply = (status: number, body: ApiBody) =>
    new Response(JSON.stringify(body), {
      status,
      headers: { "content-type": "application/json" },
    })
  const collection = () => ({
    id: "col-1",
    title: "Holiday Binder",
    description: "",
    maxCardCount: 0,
    cardCount: 3,
    binder: null,
    slug: server.slug,
  })
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input))
      const method = init?.method ?? "GET"
      calls.push(`${method} ${url.pathname}`)
      if (url.pathname === "/profile") {
        if (server.profileStatus !== 200)
          return reply(server.profileStatus, { detail: "no profile" })
        return reply(200, {
          data: { name: "Ash Ketchum", slug: "profile-ash" },
        })
      }
      if (url.pathname === `${COLLECTION}/share` && method === "POST") {
        server.slug = SLUG
        return reply(200, { data: collection() })
      }
      if (url.pathname === `${COLLECTION}/share` && method === "DELETE") {
        server.slug = null
        return reply(200, { data: collection() })
      }
      if (url.pathname === COLLECTION) return reply(200, { data: collection() })
      return reply(500, { detail: `unexpected ${method} ${url.pathname}` })
    })
  )
  return calls
}

function renderShare(server: Server) {
  const calls = stubServer(server)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <CollectionShare collectionId="col-1" />
    </QueryClientProvider>
  )
  return calls
}

const linkFor = () => `${window.location.origin}/s/profile-ash/${SLUG}`

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("Collection Share button", () => {
  it("turns sharing on, then shows the link to copy and the Shared label", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    const server: Server = { slug: null, profileStatus: 200 }
    const calls = renderShare(server)

    fireEvent.click(await screen.findByRole("button", { name: "Share" }))
    fireEvent.click(
      screen.getByRole("switch", { name: "Anyone with the link can view" })
    )

    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(calls).toContain(`POST ${COLLECTION}/share`)

    fireEvent.click(screen.getByRole("button", { name: "Copy" }))
    expect(writeText).toHaveBeenCalledWith(linkFor())
    expect(await screen.findByText("Copied")).toBeTruthy()

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await screen.findByRole("button", { name: "Shared" })).toBeTruthy()
  })

  it("asks first when turning sharing off, then ends it and shows Share again", async () => {
    const server: Server = { slug: SLUG, profileStatus: 200 }
    const calls = renderShare(server)

    fireEvent.click(await screen.findByRole("button", { name: "Shared" }))
    fireEvent.click(
      await screen.findByRole("switch", {
        name: "Anyone with the link can view",
      })
    )
    expect(
      await screen.findByText(/Existing links will stop working/)
    ).toBeTruthy()
    expect(calls).not.toContain(`DELETE ${COLLECTION}/share`)

    fireEvent.click(screen.getByRole("button", { name: "Stop sharing" }))

    await vi.waitFor(() =>
      expect(calls).toContain(`DELETE ${COLLECTION}/share`)
    )
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await screen.findByRole("button", { name: "Share" })).toBeTruthy()
  })

  it("shows nothing until the profile Slug is known, so no link can be wrong", async () => {
    const calls = renderShare({ slug: null, profileStatus: 500 })

    // Wait until both reads have been made, so the absence is not just early.
    await vi.waitFor(() => {
      expect(calls).toContain("GET /profile")
      expect(calls).toContain(`GET ${COLLECTION}`)
    })
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(screen.queryByRole("button", { name: /Share/ })).toBeNull()
  })
})
