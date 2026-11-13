import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"
import userEvent from "@testing-library/user-event"
import { NETWORK_ERROR } from "@/lib/collections"
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
} from "@tanstack/react-router"
import { useUser } from "@clerk/react"
import { NotFound } from "@/components/layout/not-found"
import type { TradeView } from "@/generated/models"
import { catalogCard } from "@/test-binder-api"
import type { ClerkState } from "@/test-clerk"
import { clerkUserResult } from "@/test-clerk"
import type { ClerkLike } from "@/lib/clerk-auth"
import { clerkAuth } from "@/lib/clerk-auth"
import { Route } from "./$slug"

// The Trade page with only the network and Clerk faked: the generated client,
// the query cache, the router and the session handling are the app's own.

// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@clerk/react", () =>
  import("@/test-clerk").then((m) => m.clerkModule())
)

const mockUseUser = vi.mocked(useUser)

const TRADE_PATH = "/trades/trade-1"
const PAGE = "/trades/trade-1"

const pikachu = catalogCard("pikachu", "Pikachu", "025")

function tradeView(overrides: Partial<TradeView> = {}) {
  return {
    slug: "trade-1",
    status: "open",
    ownerName: "Ash Ketchum",
    ownerItems: [],
    otherItems: [],
    isOwner: false,
    isParticipant: false,
    ...overrides,
  }
}

interface ApiReply {
  status: number
  body: object
}

// Answers the Trade's GET and join POST; every request URL is kept.
function stubTradeApi({
  get = { status: 200, body: { data: tradeView() } },
  join = { status: 409, body: { code: "trade_seat_taken" } },
  getAfterJoin,
  collectionReplies = [
    { status: 200, body: { data: [{ id: "collection-1", title: "Binder" }] } },
  ],
  entryReplies = [
    {
      status: 200,
      body: {
        data: [{ card: pikachu, quantity: 2 }],
        meta: { total: 1, page: 1, limit: 100 },
      },
    },
  ],
}: {
  get?: ApiReply
  join?: ApiReply
  getAfterJoin?: ApiReply
  collectionReplies?: ApiReply[]
  entryReplies?: ApiReply[]
} = {}) {
  const requests: string[] = []
  let collectionRequest = 0
  let entryRequest = 0
  const fetchMock = vi.fn(
    async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input))
      requests.push(
        `${init?.method ?? "GET"} ${url.pathname}${init?.method === "POST" && url.pathname === `${TRADE_PATH}/items` ? ` ${String(init.body)}` : ""}`
      )
      const reply = (r: ApiReply) =>
        new Response(JSON.stringify(r.body), {
          status: r.status,
          headers: { "content-type": "application/json" },
        })
      if (url.pathname === TRADE_PATH)
        return reply(
          getAfterJoin && requests.includes(`POST ${TRADE_PATH}/join`)
            ? getAfterJoin
            : get
        )
      if (url.pathname === `${TRADE_PATH}/join`) return reply(join)
      if (url.pathname === "/collections")
        return reply(
          collectionReplies[
            Math.min(collectionRequest++, collectionReplies.length - 1)
          ]
        )
      if (url.pathname === "/collections/collection-1/entries")
        return reply(
          entryReplies[Math.min(entryRequest++, entryReplies.length - 1)]
        )
      if (url.pathname === `${TRADE_PATH}/items` && init?.method === "POST")
        return reply({ status: 200, body: { data: tradeView() } })
      if (
        url.pathname.startsWith(`${TRADE_PATH}/items/`) &&
        init?.method === "DELETE"
      )
        return reply({ status: 200, body: { data: tradeView() } })
      return reply({
        status: 500,
        body: { detail: `unexpected ${url.pathname}` },
      })
    }
  )
  vi.stubGlobal("fetch", fetchMock)
  return requests
}

function clerkState(state: ClerkState) {
  mockUseUser.mockReturnValue(clerkUserResult(state))
}

afterEach(() => {
  cleanup()
  clerkAuth.onSessionChange(null)
  clerkAuth.publish(null)
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

// The route's own options, untyped: this memory router provides no `auth`
// context, which the route's typed context (the app's root) expects.
function pick(options: {
  loader?: unknown
  head?: unknown
  component?: unknown
}): any {
  const { loader, head, component } = options
  return { loader, head, component }
}

function deferredResponse() {
  let resolve!: (response: Response) => void
  const promise = new Promise<Response>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

async function renderTradePage(path = PAGE) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
    component: Outlet,
    notFoundComponent: NotFound,
  })
  const trade = createRoute({
    getParentRoute: () => root,
    path: "/trades/$slug",
    ...pick(Route.options),
  })
  const router = createRouter({
    routeTree: root.addChildren([trade]),
    history: createMemoryHistory({ initialEntries: [path] }),
    context: { queryClient },
  })
  clerkAuth.onSessionChange(() => {
    queryClient.removeQueries()
    void router.invalidate()
  })
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return router
}

describe("Trade page", () => {
  it.each(["sess_3", "sess_1", null])(
    "hides old data across stalled reads and repeated changes to %s",
    async (lastSession) => {
      clerkState("signed-in")
      const clerk = fakeClerk("sess_1")
      clerkAuth.publish(clerk)
      const stalled = deferredResponse()
      const latest = deferredResponse()
      let gets = 0
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => {
          gets += 1
          if (gets > 1) return gets === 2 ? stalled.promise : latest.promise
          return new Response(
            JSON.stringify({
              data: tradeView({
                ownerItems: [{ id: "item-1", card: pikachu, quantity: 2 }],
              }),
            }),
            {
              headers: { "content-type": "application/json" },
            }
          )
        })
      )
      await renderTradePage()
      expect(await screen.findByText("Ash Ketchum")).toBeTruthy()
      expect(screen.getByText(/Pikachu × 2/)).toBeTruthy()
      expect(
        screen.getByRole("button", { name: "Join this Trade" })
      ).toBeTruthy()

      act(() => clerk.emit("sess_2"))

      expect(screen.queryByText("Ash Ketchum")).toBeNull()
      expect(screen.queryByText(/Pikachu × 2/)).toBeNull()
      expect(
        screen.queryByRole("button", { name: "Join this Trade" })
      ).toBeNull()
      expect(screen.getByRole("status").textContent).toContain("Loading")
      await waitFor(() => expect(gets).toBe(2))
      act(() => {
        if (lastSession === null) clerkState("guest")
        clerk.emit(lastSession)
      })
      expect(screen.queryByText("Ash Ketchum")).toBeNull()
      await waitFor(() => expect(gets).toBe(3))
      await act(async () => {
        stalled.resolve(
          new Response(
            JSON.stringify({ data: tradeView({ ownerName: "Stale offer" }) })
          )
        )
      })
      expect(screen.queryByText("Stale offer")).toBeNull()
      expect(screen.queryByText("Ash Ketchum")).toBeNull()
      expect(
        screen.queryByRole("button", { name: "Join this Trade" })
      ).toBeNull()
      await act(async () => {
        latest.resolve(
          new Response(JSON.stringify({ code: "trade_not_found" }), {
            status: 404,
          })
        )
      })
      expect(
        await screen.findByRole("heading", { name: "Trade not found" })
      ).toBeTruthy()
    }
  )

  it("resets the full-seat message when navigating to another Trade", async () => {
    clerkState("signed-in")
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = new URL(String(input)).pathname
        return new Response(
          JSON.stringify(
            path.endsWith("/join")
              ? { code: "trade_seat_taken" }
              : { data: tradeView({ slug: path.split("/").at(-1) }) }
          ),
          { status: path.endsWith("/join") ? 409 : 200 }
        )
      })
    )
    const router = await renderTradePage()
    fireEvent.click(
      await screen.findByRole("button", { name: "Join this Trade" })
    )
    expect(await screen.findByText("This Trade is already full.")).toBeTruthy()

    await act(async () => {
      await router.navigate({
        to: "/trades/$slug",
        params: { slug: "trade-2" },
      })
    })

    expect(
      await screen.findByRole("button", { name: "Join this Trade" })
    ).toBeTruthy()
    expect(screen.queryByText("This Trade is already full.")).toBeNull()
  })

  it.each([true, false])(
    "clears the draft when navigating to another Trade (owner: %s)",
    async (isOwner) => {
      clerkState("signed-in")
      vi.stubGlobal(
        "fetch",
        vi.fn(async (input: RequestInfo | URL) => {
          const path = new URL(String(input)).pathname
          const data =
            path === "/collections"
              ? [{ id: "collection-1", title: "Binder" }]
              : tradeView({
                  slug: path.split("/").at(-1),
                  status: "negotiating",
                  isParticipant: true,
                  isOwner,
                })
          return new Response(JSON.stringify({ data }), {
            headers: { "content-type": "application/json" },
          })
        })
      )
      const router = await renderTradePage()
      fireEvent.change(await screen.findByLabelText("Quantity"), {
        target: { value: "7" },
      })

      await act(async () => {
        await router.navigate({
          to: "/trades/$slug",
          params: { slug: "trade-2" },
        })
      })

      expect(await screen.findByLabelText("Quantity")).toHaveProperty(
        "value",
        "1"
      )
    }
  )

  it("keeps the offer and says the Trade is full when the seat is taken", async () => {
    clerkState("signed-in")
    const requests = stubTradeApi({
      getAfterJoin: { status: 404, body: { code: "trade_not_found" } },
    })
    await renderTradePage()

    fireEvent.click(
      await screen.findByRole("button", { name: "Join this Trade" })
    )

    expect(await screen.findByText("This Trade is already full.")).toBeTruthy()
    await waitFor(() =>
      expect(requests.filter((r) => r === `GET ${TRADE_PATH}`)).toHaveLength(2)
    )
    // A non-participant keeps the offer after the authoritative read returns 404.
    expect(screen.getByText("Ash Ketchum")).toBeTruthy()
    expect(screen.queryByRole("button", { name: "Join this Trade" })).toBeNull()
    expect(
      screen.queryByRole("heading", { name: "Trade not found" })
    ).toBeNull()
  })

  it("shows the joined Trade when this participant already took the seat", async () => {
    clerkState("signed-in")
    stubTradeApi({
      getAfterJoin: {
        status: 200,
        body: {
          data: tradeView({
            status: "negotiating",
            otherName: "Misty",
            isParticipant: true,
          }),
        },
      },
    })
    await renderTradePage()

    fireEvent.click(
      await screen.findByRole("button", { name: "Join this Trade" })
    )

    expect(await screen.findByText("Negotiating")).toBeTruthy()
    expect(screen.getByText("Misty")).toBeTruthy()
    expect(screen.queryByText("This Trade is already full.")).toBeNull()
    expect(screen.queryByRole("button", { name: "Join this Trade" })).toBeNull()
  })

  it.each([
    { stage: "join", failure: "HTTP", message: "Join failed." },
    { stage: "join", failure: "network", message: NETWORK_ERROR },
    { stage: "read", failure: "HTTP", message: "Read failed." },
    { stage: "read", failure: "network", message: NETWORK_ERROR },
  ])(
    "reports a $stage $failure failure without losing the offer",
    async ({ stage, failure, message }) => {
      clerkState("signed-in")
      const errorToast = vi.spyOn(toast, "error")
      stubTradeApi()
      await renderTradePage()
      const join = await screen.findByRole("button", {
        name: "Join this Trade",
      })
      const network = vi.mocked(fetch)
      if (stage === "read")
        network.mockResolvedValueOnce(
          new Response(JSON.stringify({ code: "trade_seat_taken" }), {
            status: 409,
          })
        )
      if (failure === "network")
        network.mockRejectedValueOnce(new TypeError("offline"))
      else
        network.mockResolvedValueOnce(
          new Response(JSON.stringify({ detail: message }), { status: 500 })
        )

      fireEvent.click(join)

      await waitFor(() => expect(errorToast).toHaveBeenCalledWith(message))
      expect(screen.getByText("Ash Ketchum")).toBeTruthy()
      if (stage === "read")
        expect(screen.getByText("This Trade is already full.")).toBeTruthy()
      else
        expect(
          screen.getByRole("button", { name: "Join this Trade" })
        ).toBeTruthy()
      expect(
        screen.queryByRole("heading", { name: "Trade not found" })
      ).toBeNull()
    }
  )

  it.each([200, 404, 500, "network"] as const)(
    "ignores a late conflict read returning %s after the session changes away and back",
    async (status) => {
      clerkState("signed-in")
      const clerk = fakeClerk("sess_1")
      clerkAuth.publish(clerk)
      const errorToast = vi.spyOn(toast, "error")
      stubTradeApi()
      await renderTradePage()
      const join = await screen.findByRole("button", {
        name: "Join this Trade",
      })
      let resolve!: (response: Response) => void
      let reject!: (error: Error) => void
      const stalled = new Promise<Response>((done, fail) => {
        resolve = done
        reject = fail
      })
      vi.mocked(fetch)
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: "trade_seat_taken" }), {
            status: 409,
          })
        )
        .mockReturnValueOnce(stalled)
      fireEvent.click(join)
      await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3))

      act(() => {
        clerk.emit("sess_2")
        clerk.emit("sess_1")
      })
      expect(
        await screen.findByRole("button", { name: "Join this Trade" })
      ).toBeTruthy()
      const readsBeforeReply = vi.mocked(fetch).mock.calls.length
      await act(async () => {
        if (status === "network") reject(new TypeError("offline"))
        else
          resolve(
            new Response(
              JSON.stringify(
                status === 200
                  ? {
                      data: tradeView({
                        status: "negotiating",
                        isParticipant: true,
                      }),
                    }
                  : { detail: "Old session failure." }
              ),
              { status }
            )
          )
      })

      expect(screen.getByText("Ash Ketchum")).toBeTruthy()
      expect(
        screen.getByRole("button", { name: "Join this Trade" })
      ).toBeTruthy()
      expect(screen.queryByText("Negotiating")).toBeNull()
      expect(screen.queryByText("This Trade is already full.")).toBeNull()
      expect(errorToast).not.toHaveBeenCalled()
      expect(fetch).toHaveBeenCalledTimes(readsBeforeReply)
    }
  )

  it("adds a Card from one of the participant's Collections while negotiating", async () => {
    clerkState("signed-in")
    const requests = stubTradeApi({
      get: {
        status: 200,
        body: {
          data: tradeView({
            status: "negotiating",
            isParticipant: true,
            isOwner: true,
          }),
        },
      },
    })
    await renderTradePage()
    const user = userEvent.setup()

    await screen.findByRole("option", { name: "Binder" })
    await user.selectOptions(
      await screen.findByLabelText("Source Collection"),
      "collection-1"
    )
    await waitFor(() =>
      expect(requests).toContain("GET /collections/collection-1/entries")
    )
    fireEvent.change(await screen.findByLabelText("Card"), {
      target: { value: "pikachu" },
    })
    fireEvent.change(screen.getByLabelText("Quantity"), {
      target: { value: "3" },
    })
    fireEvent.click(screen.getByRole("button", { name: "Add Card" }))

    await waitFor(() =>
      expect(requests).toContain(
        `POST ${TRADE_PATH}/items ${JSON.stringify({ cardId: "pikachu", quantity: 3, sourceCollectionId: "collection-1" })}`
      )
    )
  })

  it("can retry failed Collection and Card loads", async () => {
    clerkState("signed-in")
    stubTradeApi({
      get: {
        status: 200,
        body: {
          data: tradeView({
            status: "negotiating",
            isParticipant: true,
            isOwner: true,
          }),
        },
      },
      collectionReplies: [
        { status: 503, body: { detail: "Unavailable" } },
        {
          status: 200,
          body: { data: [{ id: "collection-1", title: "Binder" }] },
        },
      ],
      entryReplies: [
        { status: 503, body: { detail: "Unavailable" } },
        {
          status: 200,
          body: {
            data: [{ card: pikachu, quantity: 2 }],
            meta: { total: 1, page: 1, limit: 100 },
          },
        },
      ],
    })
    await renderTradePage()
    const user = userEvent.setup()

    await screen.findByText("Could not load your Collections.")
    await user.click(screen.getByRole("button", { name: "Retry Collections" }))
    await screen.findByRole("option", { name: "Binder" })
    await user.selectOptions(
      screen.getByLabelText("Source Collection"),
      "collection-1"
    )
    await screen.findByText("Could not load Cards from this Collection.")
    await user.click(screen.getByRole("button", { name: "Retry Cards" }))

    expect(
      await screen.findByRole("option", {
        name: "Pikachu — Base Set — Rare",
      })
    ).toBeTruthy()
  })

  it("removes the participant's own Trade Item while negotiating", async () => {
    clerkState("signed-in")
    const requests = stubTradeApi({
      get: {
        status: 200,
        body: {
          data: tradeView({
            status: "negotiating",
            isParticipant: true,
            isOwner: true,
            ownerItems: [{ id: "item-1", card: pikachu, quantity: 2 }],
          }),
        },
      },
    })
    await renderTradePage()

    fireEvent.click(
      await screen.findByRole("button", { name: "Remove Pikachu" })
    )

    await waitFor(() =>
      expect(requests).toContain(`DELETE ${TRADE_PATH}/items/item-1`)
    )
  })

  it("names the item without a source Collection when it was deleted", async () => {
    clerkState("signed-in")
    stubTradeApi({
      get: {
        status: 200,
        body: {
          data: tradeView({
            isParticipant: true,
            ownerItems: [
              { id: "item-1", card: pikachu, quantity: 2 },
              {
                id: "item-2",
                card: pikachu,
                quantity: 1,
                collection: "Binder",
              },
            ],
          }),
        },
      },
    })
    await renderTradePage()

    expect(await screen.findByText("Pikachu × 2")).toBeTruthy()
    expect(screen.getByText("From a deleted Collection")).toBeTruthy()
    expect(screen.getByText("Pikachu × 1")).toBeTruthy()
    expect(screen.getByText("From Binder")).toBeTruthy()
    expect(screen.getAllByText("Base Set · Rare")).toHaveLength(2)
  })

  it("offers a Guest sign-in that returns to the Trade when the read is missing", async () => {
    clerkState("guest")
    stubTradeApi({
      get: { status: 404, body: { code: "trade_not_found" } },
    })
    await renderTradePage()

    expect(
      await screen.findByRole("heading", { name: "Trade not found" })
    ).toBeTruthy()
    const link = screen.getByRole("link", { name: "Sign in" })
    expect(link.getAttribute("href")).toContain(
      `redirect=${encodeURIComponent(PAGE)}`
    )
  })

  it("shows a signed-in reader the same not-found with no sign-in link", async () => {
    clerkState("signed-in")
    stubTradeApi({
      get: { status: 404, body: { code: "trade_not_found" } },
    })
    await renderTradePage()

    expect(
      await screen.findByRole("heading", { name: "Trade not found" })
    ).toBeTruthy()
    expect(screen.queryByRole("link", { name: "Sign in" })).toBeNull()
  })
})

function fakeClerk(sessionId: string | null): ClerkLike & {
  emit: (id: string | null) => void
} {
  const listeners = new Set<Parameters<ClerkLike["addListener"]>[0]>()
  return {
    session: sessionId ? { id: sessionId, getToken: async () => "jwt" } : null,
    addListener(callback) {
      listeners.add(callback)
      callback({ session: this.session })
      return () => listeners.delete(callback)
    },
    emit(id) {
      this.session = id ? { id, getToken: async () => "jwt" } : null
      for (const callback of listeners) callback({ session: this.session })
    },
    signOut: async () => {},
  }
}

describe("Trade loader", () => {
  it("drops a response even when the session switched away and back before it arrived", async () => {
    const clerk = fakeClerk("sess_1")
    clerkAuth.publish(clerk)
    let gets = 0
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        gets += 1
        if (gets === 1) {
          clerk.emit("sess_2")
          clerk.emit("sess_1")
        }
        return new Response(
          JSON.stringify(
            gets === 1 ? { data: tradeView() } : { code: "trade_not_found" }
          ),
          { status: gets === 1 ? 200 : 404 }
        )
      })
    )
    // SAFETY: the loader only reads queryClient and slug.
    const loader = Route.options.loader as any
    const result = await loader({
      context: {
        queryClient: new QueryClient({
          defaultOptions: { queries: { retry: false } },
        }),
      },
      params: { slug: "trade-1" },
    })

    expect(result).toMatchObject({ trade: "missing" })
  })

  it.each(["sess_3", "sess_1"])(
    "rejects the retry too when the session changes again to %s",
    async (lastSession) => {
      const clerk = fakeClerk("sess_1")
      clerkAuth.publish(clerk)
      let gets = 0
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => {
          gets += 1
          clerk.emit(gets === 1 ? "sess_2" : lastSession)
          return new Response(JSON.stringify({ data: tradeView() }))
        })
      )
      // SAFETY: the loader only reads queryClient and slug.
      const loader = Route.options.loader as any
      const result = await loader({
        context: {
          queryClient: new QueryClient({
            defaultOptions: { queries: { retry: false } },
          }),
        },
        params: { slug: "trade-1" },
      })

      expect(gets).toBe(2)
      expect(result).toMatchObject({ trade: null })
    }
  )

  it("reads again when the session changed while the request was in flight", async () => {
    clerkAuth.publish(fakeClerk("sess_1"))
    const queryClient = new QueryClient()
    let gets = 0
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        gets += 1
        // The session is replaced while this answer is still on its way.
        if (gets === 1) clerkAuth.publish(fakeClerk("sess_2"))
        return new Response(
          JSON.stringify(
            gets === 1 ? { data: tradeView() } : { code: "trade_not_found" }
          ),
          {
            status: gets === 1 ? 200 : 404,
            headers: { "content-type": "application/json" },
          }
        )
      })
    )

    // SAFETY: the loader only reads `context.queryClient` and `params.slug`.
    const loader = Route.options.loader as any
    const result = await loader({
      context: { queryClient },
      params: { slug: "trade-1" },
    })

    expect(gets).toBe(2)
    expect(result).toMatchObject({ trade: "missing" })
  })
})
