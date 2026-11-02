import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { WishlistShare } from "@/components/wishlist/wishlist-share"

// The Wishlist header's Share button against a faked network. The API has no
// `shared` flag to read, so whether it is Shared is the Shared Wishlist's own
// answer for the owner's Slug. The generated client, the query cache and the
// mutations are the app's own.

const SHARE = "/wishlist/share"
const SHARED = "/shared/profile-ash/wishlist"

// The JSON bodies the API answers with: an envelope, or an error detail.
interface ApiBody {
  data?: unknown
  detail?: string
}

interface Server {
  shared: boolean
  // "loading" never answers; "offline" fails the request outright.
  profileStatus: number | "loading" | "offline"
  // Answers the flag read with this status instead of the flag's own answer.
  flagStatus?: number | "loading"
  // Answers a share or an unshare with this status instead of accepting it.
  changeStatus?: number
  // Set to hold the flag read until released.
  holdFlagRead?: Promise<void>
}

// Each request is answered by path and method, and the calls made are kept.
function stubServer(server: Server) {
  const calls: string[] = []
  const reply = (status: number, body: ApiBody) =>
    new Response(JSON.stringify(body), {
      status,
      headers: { "content-type": "application/json" },
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
      if (url.pathname === SHARE && method === "POST") {
        if (server.changeStatus) return reply(server.changeStatus, {})
        server.shared = true
        return reply(200, { data: { slug: "profile-ash" } })
      }
      if (url.pathname === SHARE && method === "DELETE") {
        if (server.changeStatus) return reply(server.changeStatus, {})
        // An unshare answers with no content, and a 204 carries no body.
        server.shared = false
        return new Response(null, { status: 204 })
      }
      if (url.pathname === SHARED) {
        if (server.flagStatus === "loading")
          return new Promise<Response>(() => {})
        await server.holdFlagRead
        if (server.flagStatus) return reply(server.flagStatus, { detail: "no" })
        return server.shared
          ? reply(200, { data: { ownerName: "Ash Ketchum" } })
          : reply(404, { detail: "not found" })
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
      <WishlistShare />
    </QueryClientProvider>
  )
  return calls
}

const linkFor = () => `${window.location.origin}/s/profile-ash/wishlist`

// The Switch is a span, so a disabled one shows it as an attribute.
const isDisabled = (el: HTMLElement) =>
  el.hasAttribute("data-disabled") ||
  el.getAttribute("aria-disabled") === "true"

// Share is disabled until the flag is known, and a click before that does nothing.
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

describe("Wishlist Share button", () => {
  it("turns sharing on, then shows the link to copy and the Shared label", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    const calls = renderShare({ shared: false, profileStatus: 200 })

    fireEvent.click(await enabledButton("Share"))
    fireEvent.click(
      screen.getByRole("switch", { name: "Anyone with the link can view" })
    )

    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(calls).toContain(`POST ${SHARE}`)

    fireEvent.click(screen.getByRole("button", { name: "Copy" }))
    expect(writeText).toHaveBeenCalledWith(linkFor())
    expect(await screen.findByText("Copied")).toBeTruthy()

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await screen.findByRole("button", { name: "Shared" })).toBeTruthy()
  })

  it("turns sharing off at once, after asking, and shows Share again", async () => {
    const calls = renderShare({ shared: true, profileStatus: 200 })

    fireEvent.click(await enabledButton("Shared"))
    fireEvent.click(
      await screen.findByRole("switch", {
        name: "Anyone with the link can view",
      })
    )
    expect(
      await screen.findByText(/Existing links will stop working/)
    ).toBeTruthy()
    expect(calls).not.toContain(`DELETE ${SHARE}`)

    fireEvent.click(screen.getByRole("button", { name: "Stop sharing" }))

    await vi.waitFor(() => expect(calls).toContain(`DELETE ${SHARE}`))
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await screen.findByRole("button", { name: "Share" })).toBeTruthy()
  })

  it("keeps the switch pending until the shared Wishlist is read back", async () => {
    const server: Server = { shared: false, profileStatus: 200 }
    renderShare(server)
    const switchOf = () =>
      screen.getByRole("switch", { name: "Anyone with the link can view" })

    fireEvent.click(await enabledButton("Share"))
    let releaseRead = () => {}
    server.holdFlagRead = new Promise<void>((resolve) => {
      releaseRead = resolve
    })
    fireEvent.click(switchOf())

    await vi.waitFor(() => expect(isDisabled(switchOf())).toBe(true))

    releaseRead()
    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(isDisabled(switchOf())).toBe(false)
  })

  it("keeps Share visible but disabled while the flag loads", async () => {
    renderShare({ shared: true, profileStatus: 200, flagStatus: "loading" })

    const share = await screen.findByRole("button", { name: "Share" })

    expect(share).toHaveProperty("disabled", true)
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("shows a failed profile inline, and Retry loads it again", async () => {
    const server: Server = { shared: false, profileStatus: 429 }
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

  it("shows the error inline when sharing is refused", async () => {
    renderShare({ shared: false, profileStatus: 200, changeStatus: 500 })

    fireEvent.click(await enabledButton("Share"))
    fireEvent.click(
      screen.getByRole("switch", { name: "Anyone with the link can view" })
    )

    expect(
      await within(await screen.findByRole("alert")).findByText(
        "Could not share your Wishlist."
      )
    ).toBeTruthy()
  })
})
