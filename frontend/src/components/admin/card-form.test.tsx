import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MAX_IMAGE_BYTES } from "@/lib/admin-card-upload"
import { CardForm } from "./card-form"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

// jsdom decodes no images and makes no object URLs, so the browser's decoder
// reports `size`, and previews are fake addresses.
let decoded = { width: 720, height: 1005 }
beforeEach(() => {
  decoded = { width: 720, height: 1005 }
  vi.stubGlobal(
    "createImageBitmap",
    vi.fn(async () => ({ ...decoded, close: () => {} }))
  )
  URL.createObjectURL = vi.fn(() => "blob:preview")
  URL.revokeObjectURL = vi.fn()
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

const pickImage = (file = new File(["x"], "card.png", { type: "image/png" })) =>
  fireEvent.change(screen.getByLabelText("Image"), {
    target: { files: [file] },
  })

/** Picks an image the checks accept and waits for its preview. */
async function chooseImage(file?: File) {
  pickImage(file)
  await screen.findByRole("img", { name: "Preview of the chosen image" })
}

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
    await chooseImage()
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
    await chooseImage()
    save()

    await screen.findByText(/Saved SM-P 003/)
    expect(JSON.parse(String(calls[0].body))).toEqual({
      expansionSetId: "set-1",
      localId: "003",
      name: "Pikachu",
      category: "Pokémon",
    })
  })

  it.each([
    [
      "a wrong type",
      () => new File(["x"], "card.gif", { type: "image/gif" }),
      /JPEG, PNG or WebP/,
    ],
    [
      "an oversize file",
      () => {
        const file = new File(["x"], "card.png", { type: "image/png" })
        Object.defineProperty(file, "size", { value: MAX_IMAGE_BYTES + 1 })
        return file
      },
      /at most 10 MB/,
    ],
    [
      "wrong dimensions",
      () => {
        decoded = { width: 1005, height: 720 }
        return new File(["x"], "card.png", { type: "image/png" })
      },
      /portrait/,
    ],
  ])("rejects %s before anything is sent", async (_name, file, reason) => {
    const calls = stubApi(() => json(200, { data: { id: "c1" } }))
    await renderForm()
    fill({
      "Expansion set": "set-1",
      "Collector number": "005",
      Name: "Pikachu",
    })

    pickImage(file())

    expect((await screen.findByRole("alert")).textContent).toMatch(reason)
    expect(screen.queryByRole("img")).toBeNull()
    save()
    await screen.findByText(/cannot be saved without one/)
    expect(calls).toEqual([])
  })

  it("previews the chosen image", async () => {
    stubApi(() => json(200, { data: { id: "c1" } }))
    await renderForm()

    await chooseImage()

    expect(
      screen
        .getByRole("img", { name: "Preview of the chosen image" })
        .getAttribute("src")
    ).toBe("blob:preview")
  })

  it("saves and moves to the next number in the same set", async () => {
    const calls = stubApi(() =>
      json(200, { data: { id: "c1", imageUrl: "https://img.test/c1" } })
    )
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "009",
      Name: "Pikachu",
    })
    await chooseImage()
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByText(/Saved SM-P 009/)
    expect(calls.map((c) => c.path)).toContain("/admin/cards/c1/image-confirm")
    expect(
      screen.getByLabelText<HTMLSelectElement>("Expansion set").value
    ).toBe("set-1")
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("010")
    expect(screen.getByLabelText<HTMLInputElement>("Name").value).toBe("")
    expect(screen.queryByRole("img")).toBeNull()
    fill({ Name: "Raichu" })
    save()
    await screen.findByText(/cannot be saved without one/)
  })

  it("carries the number past its padding", async () => {
    stubApi(() => json(200, { data: { id: "c1" } }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "99",
      Name: "Pikachu",
    })
    await chooseImage()
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByText(/Saved SM-P 99/)
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("100")
  })

  it("keeps the form as it is when a save and next fails", async () => {
    stubApi(() => json(422, { status: 422, detail: "not an image" }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "009",
      Name: "Pikachu",
    })
    await chooseImage()
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByRole("alert")
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("009")
    expect(screen.getByLabelText<HTMLInputElement>("Name").value).toBe(
      "Pikachu"
    )
  })

  it("shows why a refused image was not saved", async () => {
    stubApi(() => json(422, { status: 422, detail: "not an image" }))
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "004",
      Name: "Pikachu",
    })
    await chooseImage()
    save()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /not an image/
    )
  })
})
