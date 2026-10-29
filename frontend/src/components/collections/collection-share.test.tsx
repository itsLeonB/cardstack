import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react"
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
  // "loading" never answers; "offline" fails the request outright.
  profileStatus: number | "loading" | "offline"
  // Set to hold the Collection read-back until released.
  holdCollectionRead?: Promise<void>
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
        if (server.profileStatus === "loading")
          return new Promise<Response>(() => {})
        if (server.profileStatus === "offline")
          throw new TypeError("Failed to fetch")
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
      if (url.pathname === COLLECTION) {
        await server.holdCollectionRead
        return reply(200, { data: collection() })
      }
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

// The Switch is a span, so a disabled one shows it as an attribute.
const isDisabled = (el: HTMLElement) =>
  el.hasAttribute("data-disabled") ||
  el.getAttribute("aria-disabled") === "true"

// Share is disabled until the profile loads, and a click before that does nothing.
async function enabledButton(name: string) {
  await vi.waitFor(() =>
    expect(screen.getByRole("button", { name })).toHaveProperty(
      "disabled",
      false
    )
  )
  return screen.getByRole("button", { name })
}

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

    fireEvent.click(await enabledButton("Share"))
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

    fireEvent.click(await enabledButton("Shared"))
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

  it("keeps the switch pending until the shared Collection is read back", async () => {
    const server: Server = { slug: null, profileStatus: 200 }
    const calls = renderShare(server)
    const switchOf = () =>
      screen.getByRole("switch", { name: "Anyone with the link can view" })

    fireEvent.click(await enabledButton("Share"))
    let releaseRead = () => {}
    server.holdCollectionRead = new Promise<void>((resolve) => {
      releaseRead = resolve
    })
    fireEvent.click(switchOf())

    await vi.waitFor(() =>
      expect(calls.filter((call) => call === `GET ${COLLECTION}`)).toHaveLength(
        2
      )
    )
    expect(isDisabled(switchOf())).toBe(true)

    releaseRead()
    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(isDisabled(switchOf())).toBe(false)
  })

  it("keeps the switch pending until the unshared Collection is read back", async () => {
    const server: Server = { slug: SLUG, profileStatus: 200 }
    const calls = renderShare(server)
    const switchOf = () =>
      screen.getByRole("switch", { name: "Anyone with the link can view" })

    fireEvent.click(await enabledButton("Shared"))
    let releaseRead = () => {}
    server.holdCollectionRead = new Promise<void>((resolve) => {
      releaseRead = resolve
    })
    fireEvent.click(
      await screen.findByRole("switch", {
        name: "Anyone with the link can view",
      })
    )
    fireEvent.click(screen.getByRole("button", { name: "Stop sharing" }))

    await vi.waitFor(() =>
      expect(calls.filter((call) => call === `GET ${COLLECTION}`)).toHaveLength(
        2
      )
    )
    expect(isDisabled(switchOf())).toBe(true)

    releaseRead()
    await vi.waitFor(() => expect(isDisabled(switchOf())).toBe(false))
    expect(switchOf().getAttribute("aria-checked")).toBe("false")
  })

  it("keeps Share visible but disabled while the profile loads", async () => {
    renderShare({ slug: null, profileStatus: "loading" })

    const share = await screen.findByRole("button", { name: "Share" })

    expect(share).toHaveProperty("disabled", true)
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("shows a failed profile inline, and Retry loads it again", async () => {
    const server: Server = { slug: null, profileStatus: 429 }
    const calls = renderShare(server)

    const alert = await screen.findByRole("alert")
    expect(within(alert).getByText("Could not load sharing")).toBeTruthy()
    expect(screen.getByRole("button", { name: "Share" })).toHaveProperty(
      "disabled",
      true
    )

    server.profileStatus = 200
    fireEvent.click(within(alert).getByRole("button", { name: "Retry" }))

    await vi.waitFor(() => expect(screen.queryByRole("alert")).toBeNull())
    expect(calls.filter((call) => call === "GET /profile")).toHaveLength(2)
    await enabledButton("Share")
  })

  it("does not open a Shared Collection's dialog while its profile is missing", async () => {
    renderShare({ slug: SLUG, profileStatus: "offline" })

    const shared = await screen.findByRole("button", { name: "Shared" })
    await screen.findByRole("alert")
    expect(shared).toHaveProperty("disabled", true)

    fireEvent.click(shared)
    expect(screen.queryByRole("dialog")).toBeNull()
  })
})
