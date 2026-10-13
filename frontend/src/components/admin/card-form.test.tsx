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

interface Stub {
  upsert?: () => Response
  lookup?: () => Response
  confirm?: () => Response
}

/** The API as the form meets it: lookups, the upsert and the upload calls. */
function stubApi({ upsert, lookup, confirm }: Stub = {}) {
  const calls: Call[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const path = url === SIGNED_URL ? "bucket" : new URL(url).pathname
      const method = init?.method ?? "GET"
      if (method !== "GET" || path === "/admin/cards") {
        calls.push({ method, path, body: init?.body })
      }

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
      if (path === "/admin/cards" && method === "GET") {
        return lookup ? lookup() : json(200, { data: { id: "c1" } })
      }
      if (path === "/admin/cards") {
        return upsert ? upsert() : json(200, { data: { id: "c1" } })
      }
      if (path.endsWith("/image-upload")) {
        return json(200, {
          data: { url: SIGNED_URL, contentType: "image/png", expiresAt: "x" },
        })
      }
      if (path === "bucket") return new Response(null, { status: 200 })
      return confirm ? confirm() : json(200, { data: { id: "c1" } })
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
const upload = () =>
  fireEvent.click(screen.getByRole("button", { name: "Upload image" }))

const saved = (imageUrl?: string) => json(200, { data: { id: "c1", imageUrl } })

describe("CardForm details", () => {
  it("saves a card without an image and says it stays hidden", async () => {
    const calls = stubApi({ upsert: () => saved() })
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "002",
      Name: "Professor's Research",
      Type: "Supporter",
      Rarity: "r-2",
    })
    save()

    expect((await screen.findByRole("status")).textContent).toMatch(
      /Saved SM-P 002.*hidden from the catalog until an image is uploaded/
    )
    expect(calls.map((c) => [c.method, c.path])).toEqual([
      ["PUT", "/admin/cards"],
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

  it("does not call a card hidden once it has an image", async () => {
    stubApi({ upsert: () => saved("https://img.test/c1") })
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "002",
      Name: "Fixed name",
    })
    save()

    const status = await screen.findByRole("status")
    expect(status.textContent).toMatch(/Saved SM-P 002/)
    expect(status.textContent).not.toMatch(/hidden/)
  })

  it("leaves the tag and rarity out for a Pokémon on the default rarity", async () => {
    const calls = stubApi()
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "003",
      Name: "Pikachu",
    })
    save()

    await screen.findByText(/Saved SM-P 003/)
    expect(JSON.parse(String(calls[0].body))).toEqual({
      expansionSetId: "set-1",
      localId: "003",
      name: "Pikachu",
      category: "Pokémon",
    })
  })

  it("asks for a set, a number and a name before sending anything", async () => {
    const calls = stubApi()
    await renderForm()

    save()

    expect((await screen.findByRole("alert")).textContent).toMatch(/name/i)
    expect(calls).toEqual([])
  })

  it("saves and moves to the next number in the same set", async () => {
    const calls = stubApi()
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "009",
      Name: "Pikachu",
    })
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByText(/Saved SM-P 009/)
    expect(calls.map((c) => c.path)).toEqual(["/admin/cards"])
    expect(
      screen.getByLabelText<HTMLSelectElement>("Expansion set").value
    ).toBe("set-1")
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("010")
    expect(screen.getByLabelText<HTMLInputElement>("Name").value).toBe("")
  })

  it("carries the number past its padding", async () => {
    stubApi()
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "99",
      Name: "Pikachu",
    })
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByText(/Saved SM-P 99/)
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("100")
  })

  it("keeps the form as it is when a save and next fails", async () => {
    stubApi({ upsert: () => json(422, { status: 422, detail: "bad rarity" }) })
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "009",
      Name: "Pikachu",
    })
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    expect((await screen.findByRole("alert")).textContent).toMatch(/bad rarity/)
    expect(
      screen.getByLabelText<HTMLInputElement>("Collector number").value
    ).toBe("009")
    expect(screen.getByLabelText<HTMLInputElement>("Name").value).toBe(
      "Pikachu"
    )
  })

  it("forgets the chosen image when it moves to the next number", async () => {
    stubApi()
    await renderForm()

    fill({
      "Expansion set": "set-1",
      "Collector number": "009",
      Name: "Pikachu",
    })
    await chooseImage()
    fireEvent.click(screen.getByRole("button", { name: "Save and next" }))

    await screen.findByText(/Saved SM-P 009/)
    expect(screen.queryByRole("img")).toBeNull()
    upload()
    expect((await screen.findByRole("alert")).textContent).toMatch(
      /Choose an image file/
    )
  })
})

describe("CardForm image", () => {
  it("does not upload without an image file", async () => {
    const calls = stubApi()
    await renderForm()

    fill({ "Expansion set": "set-1", "Collector number": "001" })
    upload()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /Choose an image file/
    )
    expect(calls).toEqual([])
  })

  it("uploads the image for the card at the set and number in the form", async () => {
    const calls = stubApi()
    await renderForm()

    fill({ "Expansion set": "set-1", "Collector number": "002" })
    await chooseImage()
    upload()

    await screen.findByText(/Uploaded the image of SM-P 002/)
    expect(calls.map((c) => [c.method, c.path])).toEqual([
      ["GET", "/admin/cards"],
      ["POST", "/admin/cards/c1/image-upload"],
      ["PUT", "bucket"],
      ["POST", "/admin/cards/c1/image-confirm"],
    ])
    expect(screen.queryByRole("img")).toBeNull()
  })

  it("asks for the card's details first when there is no such card", async () => {
    const calls = stubApi({
      lookup: () => json(404, { status: 404, detail: "card not found" }),
    })
    await renderForm()

    fill({ "Expansion set": "set-1", "Collector number": "002" })
    await chooseImage()
    upload()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /no card SM-P 002 yet.*Save its details first/
    )
    expect(calls.map((c) => c.path)).toEqual(["/admin/cards"])
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
    const calls = stubApi()
    await renderForm()
    fill({ "Expansion set": "set-1", "Collector number": "005" })

    pickImage(file())

    expect((await screen.findByRole("alert")).textContent).toMatch(reason)
    expect(screen.queryByRole("img")).toBeNull()
    upload()
    await screen.findByText(/Choose an image file/)
    expect(calls).toEqual([])
  })

  it("previews the chosen image", async () => {
    stubApi()
    await renderForm()

    await chooseImage()

    expect(
      screen
        .getByRole("img", { name: "Preview of the chosen image" })
        .getAttribute("src")
    ).toBe("blob:preview")
  })

  it("shows why a refused image was not accepted", async () => {
    stubApi({
      confirm: () => json(422, { status: 422, detail: "not an image" }),
    })
    await renderForm()

    fill({ "Expansion set": "set-1", "Collector number": "004" })
    await chooseImage()
    upload()

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /not an image/
    )
  })
})
