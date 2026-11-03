import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { WishlistShare } from "@/components/wishlist/wishlist-share"

// The Wishlist header's Share button against a faked network: the shared flag
// is the server's state, so a share or an unshare only shows once the flag is
// read back. The generated client, the query cache and the mutations are the
// app's own.

const PROFILE_ENDPOINT = "/profile"
const META_ENDPOINT = "/wishlist/meta"
const SHARE_ENDPOINT = "/wishlist/share"
const SWITCH_NAME = "Anyone with the link can view"

// The JSON bodies the API answers with: an envelope, or an error detail.
interface ApiBody {
  data?: unknown
  detail?: string
}

interface Server {
  // The flag the meta endpoint reports, which sharing changes.
  shared: boolean
  // "loading" never answers; "offline" fails the flag read outright.
  metaStatus?: "loading" | "offline"
  // Set to hold the flag read-back until released.
  holdMetaRead?: Promise<void>
  // Answers a share with this status instead of accepting it.
  shareStatus?: number
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

      if (url.pathname === PROFILE_ENDPOINT) {
        return reply(200, {
          data: { slug: "profile-ash", name: "Ash Ketchum" },
        })
      }
      if (url.pathname === META_ENDPOINT && method === "GET") {
        if (server.metaStatus === "loading")
          return new Promise<Response>(() => {})
        if (server.metaStatus === "offline")
          throw new TypeError("Failed to fetch")
        await server.holdMetaRead
        return reply(200, { data: { shared: server.shared } })
      }
      if (url.pathname === SHARE_ENDPOINT && method === "POST") {
        if (server.shareStatus)
          return reply(server.shareStatus, { detail: "Not allowed" })
        server.shared = true
        return reply(200, { data: { slug: "profile-ash" } })
      }
      if (url.pathname === SHARE_ENDPOINT && method === "DELETE") {
        // An unshare answers with no content, and a 204 carries no body.
        server.shared = false
        return new Response(null, { status: 204 })
      }
      return reply(500, { detail: `unexpected ${method} ${url.pathname}` })
    })
  )
  return calls
}

function renderWishlistShare(server: Server) {
  const calls = stubServer(server)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <WishlistShare />
    </QueryClientProvider>
  )
  return calls
}

const linkFor = () => `${window.location.origin}/s/profile-ash/wishlist`

// The label follows the flag, so a test waits for the label the answer should
// give. Matching the whole name keeps "Share" and "Shared" apart.
const buttonFor = (label: "Share" | "Shared") =>
  screen.findByRole("button", { name: label })

// The control is disabled until the flag is known, and a click before that does
// nothing, so a test that acts on it waits for the enabled one.
async function enabledButton(label: "Share" | "Shared") {
  await waitFor(() =>
    expect(screen.getByRole("button", { name: label })).toHaveProperty(
      "disabled",
      false
    )
  )
  return screen.getByRole("button", { name: label })
}

// Queried each time, so an assertion reads the switch the dialog shows now.
const switchOf = () => screen.getByRole("switch", { name: SWITCH_NAME })

// A disabled switch shows it as an attribute.
const isDisabled = (el: HTMLElement) =>
  el.hasAttribute("data-disabled") ||
  el.getAttribute("aria-disabled") === "true"

const metaReads = (calls: string[]) =>
  calls.filter((call) => call === `GET ${META_ENDPOINT}`).length

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("Wishlist Share button", () => {
  it("names the button from the meta flag, off by default", async () => {
    const calls = renderWishlistShare({ shared: false })

    await waitFor(() => expect(calls).toContain(`GET ${META_ENDPOINT}`))
    await enabledButton("Share")
    expect(screen.queryByRole("button", { name: "Shared" })).toBeNull()
    // The flag is read from the meta endpoint, never from the public page.
    expect(calls.some((call) => call.includes("/shared/"))).toBe(false)

    cleanup()

    renderWishlistShare({ shared: true })
    await enabledButton("Shared")
    expect(screen.queryByRole("button", { name: "Share" })).toBeNull()
  })

  it("warns before the first share, and offers no link while the flag is off", async () => {
    const calls = renderWishlistShare({ shared: false })

    fireEvent.click(await enabledButton("Share"))

    expect(await screen.findByText(/full name/i)).toBeTruthy()
    expect(switchOf().getAttribute("aria-checked")).toBe("false")
    expect(screen.queryByRole("textbox", { name: "Share link" })).toBeNull()
    expect(calls).not.toContain(`POST ${SHARE_ENDPOINT}`)
  })

  it("turns sharing on, shows the link to copy, and reads the flag back", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal("navigator", { clipboard: { writeText } })
    const calls = renderWishlistShare({ shared: false })

    fireEvent.click(await enabledButton("Share"))
    fireEvent.click(await screen.findByRole("switch", { name: SWITCH_NAME }))

    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(calls).toContain(`POST ${SHARE_ENDPOINT}`)
    // The answer's flag is what the dialog shows, not the click: the warning
    // goes with it, and the flag was read again to get here.
    expect(screen.queryByText(/full name/i)).toBeNull()
    expect(metaReads(calls)).toBeGreaterThan(1)

    fireEvent.click(screen.getByRole("button", { name: "Copy" }))
    expect(writeText).toHaveBeenCalledWith(linkFor())
    expect(await screen.findByText("Copied")).toBeTruthy()

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await buttonFor("Shared")).toBeTruthy()
  })

  it("keeps the switch pending until the flag is read back", async () => {
    const server: Server = { shared: false }
    const calls = renderWishlistShare(server)

    fireEvent.click(await enabledButton("Share"))
    await screen.findByRole("switch", { name: SWITCH_NAME })
    let releaseRead = () => {}
    server.holdMetaRead = new Promise<void>((resolve) => {
      releaseRead = resolve
    })
    fireEvent.click(switchOf())

    await waitFor(() => expect(metaReads(calls)).toBe(2))
    expect(isDisabled(switchOf())).toBe(true)

    releaseRead()
    expect(
      await screen.findByRole("textbox", { name: "Share link" })
    ).toHaveProperty("value", linkFor())
    expect(isDisabled(switchOf())).toBe(false)
  })

  it("keeps Share disabled while the flag is being read", async () => {
    renderWishlistShare({ shared: true, metaStatus: "loading" })

    const share = await buttonFor("Share")
    expect(share).toHaveProperty("disabled", true)
    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("shows a failed flag read inline, and Retry reads the flag again", async () => {
    const server: Server = { shared: true, metaStatus: "offline" }
    const calls = renderWishlistShare(server)

    const alert = await screen.findByRole("alert")
    expect(within(alert).getByText("Could not load sharing")).toBeTruthy()
    // No answer, so no state to act on: the control stays disabled.
    expect(screen.getByRole("button", { name: "Share" })).toHaveProperty(
      "disabled",
      true
    )

    server.metaStatus = undefined
    fireEvent.click(within(alert).getByRole("button", { name: "Retry" }))

    await enabledButton("Shared")
    expect(screen.queryByRole("alert")).toBeNull()
    expect(metaReads(calls)).toBe(2)
  })

  it("shows why a refused share failed, and leaves the flag alone", async () => {
    const calls = renderWishlistShare({ shared: false, shareStatus: 403 })

    fireEvent.click(await enabledButton("Share"))
    fireEvent.click(await screen.findByRole("switch", { name: SWITCH_NAME }))

    expect((await screen.findByRole("alert")).textContent).toBe("Not allowed")
    expect(switchOf().getAttribute("aria-checked")).toBe("false")
    expect(screen.queryByRole("textbox", { name: "Share link" })).toBeNull()
    // A refusal is not a change, so there is no flag to read back.
    expect(metaReads(calls)).toBe(1)
  })

  it("asks before turning sharing off, and ends the link at once", async () => {
    const calls = renderWishlistShare({ shared: true })

    fireEvent.click(await enabledButton("Shared"))
    fireEvent.click(await screen.findByRole("switch", { name: SWITCH_NAME }))

    expect(
      await screen.findByText(/Existing links will stop working/)
    ).toBeTruthy()
    // Asking is not ending it: nothing has been sent, and the flag is still on.
    expect(calls).not.toContain(`DELETE ${SHARE_ENDPOINT}`)
    expect(switchOf().getAttribute("aria-checked")).toBe("true")

    fireEvent.click(screen.getByRole("button", { name: "Stop sharing" }))

    await waitFor(() =>
      expect(switchOf().getAttribute("aria-checked")).toBe("false")
    )
    expect(calls).toContain(`DELETE ${SHARE_ENDPOINT}`)
    expect(metaReads(calls)).toBeGreaterThan(1)
    expect(screen.queryByRole("textbox", { name: "Share link" })).toBeNull()
    expect(screen.getByText(/full name/i)).toBeTruthy()

    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" })
    expect(await buttonFor("Share")).toBeTruthy()
  })
})
