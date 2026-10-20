import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { DeleteCardForm } from "./delete-card-form"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

interface Call {
  method: string
  path: string
  search: string
}

function json<T>(status: number, body: T) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

interface Stub {
  lookup?: () => Response
  remove?: () => Response
}

const FOUND = () => json(200, { data: { id: "c1", name: "Pikachu" } })
const REMOVED = () => new Response(null, { status: 204 })

/** The API as the form meets it: the set list, the lookup, then the delete. */
function stubApi({ lookup = FOUND, remove = REMOVED }: Stub = {}) {
  const calls: Call[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input))
      const method = init?.method ?? "GET"
      if (url.pathname === "/catalog/series") {
        return json(200, {
          data: {
            series: [],
            ungroupedExpansionSets: [
              { id: "set-1", code: "SM-P", name: "Kartu Promo", imageUrl: "" },
            ],
          },
        })
      }
      calls.push({ method, path: url.pathname, search: url.search })
      return method === "DELETE" ? remove() : lookup()
    })
  )
  return calls
}

const deletes = (calls: Call[]) => calls.filter((c) => c.method === "DELETE")

async function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <DeleteCardForm />
    </QueryClientProvider>
  )
  await screen.findByRole("option", { name: /SM-P/ })
}

function choose(setId: string, number: string) {
  fireEvent.change(screen.getByLabelText("Expansion set"), {
    target: { value: setId },
  })
  fireEvent.change(screen.getByLabelText("Collector number"), {
    target: { value: number },
  })
}

const openConfirm = () =>
  fireEvent.click(screen.getByRole("button", { name: "Delete card" }))

/** Enters a card and waits until the lookup has named it. */
async function chooseFound(number = "007") {
  choose("set-1", number)
  await screen.findByText(/Found: Pikachu/)
}

const confirm = () =>
  fireEvent.click(
    within(screen.getByRole("alertdialog")).getByRole("button", {
      name: "Delete",
    })
  )

describe("DeleteCardForm", () => {
  it("asks for a set and a number before it offers to delete", async () => {
    const calls = stubApi()
    await renderForm()

    openConfirm()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /expansion set/i
    )
    expect(screen.queryByRole("alertdialog")).toBeNull()
    expect(calls).toEqual([])
  })

  it("looks the card up by set and number, and names it in the confirmation", async () => {
    const calls = stubApi()
    await renderForm()
    await chooseFound()

    openConfirm()

    const dialog = (await screen.findByRole("alertdialog")).textContent
    expect(dialog).toMatch(/Delete SM-P 007 Pikachu\?/)
    expect(calls).toEqual([
      {
        method: "GET",
        path: "/admin/cards",
        search: "?expansionSetId=set-1&localId=007",
      },
    ])
  })

  it("sends no delete until the deletion is confirmed", async () => {
    const calls = stubApi()
    await renderForm()
    await chooseFound()

    openConfirm()
    await screen.findByRole("alertdialog")

    expect(deletes(calls)).toEqual([])
  })

  it("sends no delete when the confirmation is cancelled", async () => {
    const calls = stubApi()
    await renderForm()
    await chooseFound()
    openConfirm()
    const dialog = await screen.findByRole("alertdialog")

    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }))

    expect(screen.queryByRole("alertdialog")).toBeNull()
    expect(deletes(calls)).toEqual([])
    expect(screen.queryByText(/Deleted SM-P/)).toBeNull()
  })

  it("deletes the card by id once confirmed", async () => {
    const calls = stubApi()
    await renderForm()
    await chooseFound()
    openConfirm()
    await screen.findByRole("alertdialog")

    confirm()

    await screen.findByText(/Deleted SM-P 007/)
    expect(deletes(calls)).toEqual([
      { method: "DELETE", path: "/admin/cards/c1", search: "" },
    ])
  })

  it("offers nothing to confirm for a card that does not exist", async () => {
    const calls = stubApi({
      lookup: () => json(404, { status: 404, detail: "card not found" }),
    })
    await renderForm()
    choose("set-1", "404")
    await screen.findByText(/There is no card there/)

    openConfirm()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /no card SM-P 404/
    )
    expect(screen.queryByRole("alertdialog")).toBeNull()
    expect(deletes(calls)).toEqual([])
  })

  it("shows why a card was not deleted", async () => {
    stubApi({
      remove: () =>
        json(409, { status: 409, detail: "card is in a collection" }),
    })
    await renderForm()
    await chooseFound()
    openConfirm()
    await screen.findByRole("alertdialog")

    confirm()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /card is in a collection/
    )
    expect(screen.queryByText(/Deleted SM-P/)).toBeNull()
  })
})
