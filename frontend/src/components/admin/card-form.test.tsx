import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CardForm } from "./card-form"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

const SIGNED_URL = "https://bucket.test/cards/c1?sig=1"

interface Call {
  method: string
  path: string
  body: unknown
}

function json<T>(status: number, body: T) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

/** The API as the form meets it: lookups, then the upsert and upload calls. */
function stubApi(confirm: () => Response) {
  const calls: Call[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const path = url === SIGNED_URL ? "bucket" : new URL(url).pathname
      const method = init?.method ?? "GET"
      if (method !== "GET") calls.push({ method, path, body: init?.body })

      if (path === "/catalog/series") {
        return json(200, {
          data: {
            series: [],
            ungroupedExpansionSets: [
              { id: "set-1", code: "SM-P", name: "Kartu Promo", imageUrl: "" },
            ],
          },
        })
      }
      if (path === "/catalog/rarities") {
        return json(200, {
          data: [{ id: "r-2", code: "RR", name: "Double Rare" }],
        })
      }
      if (path === "/admin/cards") return json(200, { data: { id: "c1" } })
      if (path.endsWith("/image-upload")) {
        return json(200, {
          data: { url: SIGNED_URL, contentType: "image/png", expiresAt: "x" },
        })
      }
      if (path === "bucket") return new Response(null, { status: 200 })
      return confirm()
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
      <CardForm />
    </QueryClientProvider>
  )
  await screen.findByRole("option", { name: /SM-P/ })
  await screen.findByRole("option", { name: "Double Rare" })
}

function fill(fields: Record<string, string>) {
  for (const [label, value] of Object.entries(fields)) {
    fireEvent.change(screen.getByLabelText(label), { target: { value } })
  }
}

const chooseImage = () =>
  fireEvent.change(screen.getByLabelText("Image"), {
    target: { files: [new File(["x"], "card.png", { type: "image/png" })] },
  })

const save = () => fireEvent.click(screen.getByRole("button", { name: "Save" }))

describe("CardForm", () => {
  it("does not save a card that has no image", async () => {
    const calls = stubApi(() => json(200, { data: { id: "c1" } }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "001",
      Name: "Pikachu",
    })
    save()

    expect((await screen.findByRole("alert")).textContent).toMatch(/image/i)
    expect(calls).toEqual([])
  })

  it("saves the card, uploads the image and reports it saved", async () => {
    const calls = stubApi(() =>
      json(200, { data: { id: "c1", imageUrl: "https://img.test/c1" } })
    )
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "002",
      Name: "Professor's Research",
      Type: "Supporter",
      Rarity: "r-2",
    })
    chooseImage()
    save()

    await screen.findByText(/Saved SM-P 002/)
    expect(calls.map((c) => [c.method, c.path])).toEqual([
      ["PUT", "/admin/cards"],
      ["POST", "/admin/cards/c1/image-upload"],
      ["PUT", "bucket"],
      ["POST", "/admin/cards/c1/image-confirm"],
    ])
    expect(JSON.parse(String(calls[0].body))).toEqual({
      expansionSetId: "set-1",
      localId: "002",
      name: "Professor's Research",
      category: "Trainer",
      tag: "Supporter",
      rarityId: "r-2",
    })
  })

  it("leaves the tag and rarity out for a Pokémon on the default rarity", async () => {
    const calls = stubApi(() => json(200, { data: { id: "c1" } }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "003",
      Name: "Pikachu",
    })
    chooseImage()
    save()

    await screen.findByText(/Saved SM-P 003/)
    expect(JSON.parse(String(calls[0].body))).toEqual({
      expansionSetId: "set-1",
      localId: "003",
      name: "Pikachu",
      category: "Pokémon",
    })
  })

  it("shows why a refused image was not saved", async () => {
    stubApi(() => json(422, { status: 422, detail: "not an image" }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "004",
      Name: "Pikachu",
    })
    chooseImage()
    save()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /not an image/
    )
  })
})
