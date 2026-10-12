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

/** The API as the form meets it: the set list, then the delete call. */
function stubApi(remove: () => Response) {
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
      return remove()
    })
  )
  return calls
}

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

const confirm = () =>
  fireEvent.click(
    within(screen.getByRole("alertdialog")).getByRole("button", {
      name: "Delete",
    })
  )

describe("DeleteCardForm", () => {
  it("asks for a set and a number before it offers to delete", async () => {
    const calls = stubApi(() => new Response(null, { status: 204 }))
    await renderForm()

    openConfirm()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /expansion set/i
    )
    expect(screen.queryByRole("alertdialog")).toBeNull()
    expect(calls).toEqual([])
  })

  it("sends nothing until the deletion is confirmed", async () => {
    const calls = stubApi(() => new Response(null, { status: 204 }))
    await renderForm()
    choose("set-1", "007")

    openConfirm()

    expect((await screen.findByRole("alertdialog")).textContent).toMatch(
      /SM-P 007/
    )
    expect(calls).toEqual([])
  })

  it("deletes the card at that set and number once confirmed", async () => {
    const calls = stubApi(() => new Response(null, { status: 204 }))
    await renderForm()
    choose("set-1", "007")
    openConfirm()
    await screen.findByRole("alertdialog")

    confirm()

    await screen.findByText(/Deleted SM-P 007/)
    expect(calls).toEqual([
      {
        method: "DELETE",
        path: "/admin/cards",
        search: "?expansionSetId=set-1&localId=007",
      },
    ])
  })

  it("shows why a card was not deleted", async () => {
    stubApi(() => json(409, { status: 409, detail: "card is in a collection" }))
    await renderForm()
    choose("set-1", "007")
    openConfirm()
    await screen.findByRole("alertdialog")

    confirm()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /card is in a collection/
    )
    expect(screen.queryByText(/Deleted SM-P/)).toBeNull()
  })
})
