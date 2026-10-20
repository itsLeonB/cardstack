import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ExpansionSetForm } from "./expansion-set-form"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

const SERIES = {
  series: [
    {
      id: "series-sm",
      code: "matahari-bulan",
      name: "Matahari & Bulan",
      imageUrl: "",
      expansionSets: [
        {
          id: "set-sm1",
          code: "SM1",
          name: "Koleksi Matahari",
          releaseDate: "2016-12-02T00:00:00Z",
          imageUrl: "",
        },
      ],
    },
  ],
  ungroupedExpansionSets: [
    {
      id: "set-loose",
      code: "LOOSE",
      name: "Set tanpa Series",
      releaseDate: "2020-01-01T00:00:00Z",
      imageUrl: "",
    },
  ],
}

interface Write {
  method: string
  path: string
  body: unknown
}

/**
 * Stubs the API: GET /catalog/series, POST /admin/expansion-sets and PUT
 * /admin/expansion-sets/{id}. `writes` records the two writes.
 */
function renderForm(writeResponse?: () => Response) {
  const writes: Write[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = new URL(String(input)).pathname
      const method = init?.method ?? "GET"
      if (method === "POST" || method === "PUT") {
        writes.push({ method, path, body: JSON.parse(String(init?.body)) })
        return writeResponse
          ? writeResponse()
          : savedResponse(method === "POST" ? 201 : 200)
      }
      return json({ data: SERIES })
    })
  )
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <ExpansionSetForm />
    </QueryClientProvider>
  )
  return writes
}

interface StubBody {
  data?: object
  status?: number
  title?: string
  detail?: string
}

function json(body: StubBody, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

function savedResponse(status: number) {
  return json({ data: { id: "x", code: "x", name: "x", imageUrl: "" } }, status)
}

function fill(label: string, value: string) {
  fireEvent.change(screen.getByLabelText(label), { target: { value } })
}

async function chooseSeries(name: string) {
  const series = await screen.findByLabelText("Series")
  await screen.findByRole("option", { name })
  fireEvent.change(series, {
    target: {
      value: screen.getByRole("option", { name }).getAttribute("value") ?? "",
    },
  })
}

function save() {
  fireEvent.click(screen.getByRole("button", { name: "Save set" }))
}

describe("ExpansionSetForm", () => {
  it("creates a set under a chosen Series", async () => {
    const puts = renderForm()

    fill("Code", "SM-P")
    fill("Name", "Kartu Promo Matahari & Bulan")
    fill("Release date", "2019-05-08")
    await chooseSeries("Matahari & Bulan")
    save()

    await screen.findByText("Saved SM-P.")
    expect(puts).toEqual([
      {
        method: "POST",
        path: "/admin/expansion-sets",
        body: {
          code: "SM-P",
          name: "Kartu Promo Matahari & Bulan",
          releaseDate: "2019-05-08",
          seriesId: "series-sm",
        },
      },
    ])
  })

  it("edits an existing set: loads its fields, keeps its code, PUTs every field by id", async () => {
    const puts = renderForm()

    await screen.findByRole("option", { name: /SM1/ })
    fireEvent.change(screen.getByLabelText("Existing set"), {
      target: { value: "set-sm1" },
    })

    expect(screen.getByLabelText("Code")).toHaveProperty("value", "SM1")
    expect(screen.getByLabelText("Code").hasAttribute("readonly")).toBe(true)
    expect(screen.getByLabelText("Name")).toHaveProperty(
      "value",
      "Koleksi Matahari"
    )
    expect(screen.getByLabelText("Release date")).toHaveProperty(
      "value",
      "2016-12-02"
    )

    fill("Name", "Koleksi Matahari (diperbaiki)")
    save()

    await screen.findByText("Saved SM1.")
    expect(puts).toEqual([
      {
        method: "PUT",
        path: "/admin/expansion-sets/set-sm1",
        body: {
          name: "Koleksi Matahari (diperbaiki)",
          releaseDate: "2016-12-02",
          seriesId: "series-sm",
        },
      },
    ])
  })

  it("lists a set with no Series and requires a Series before the PUT", async () => {
    const puts = renderForm()

    await screen.findByRole("option", { name: /LOOSE/ })
    fireEvent.change(screen.getByLabelText("Existing set"), {
      target: { value: "set-loose" },
    })
    expect(screen.getByLabelText("Name")).toHaveProperty(
      "value",
      "Set tanpa Series"
    )

    save()
    await screen.findByText("Series is required.")
    expect(puts).toEqual([])

    await chooseSeries("Matahari & Bulan")
    save()

    await screen.findByText("Saved LOOSE.")
    expect(puts).toEqual([
      {
        method: "PUT",
        path: "/admin/expansion-sets/set-loose",
        body: {
          name: "Set tanpa Series",
          releaseDate: "2020-01-01",
          seriesId: "series-sm",
        },
      },
    ])
  })

  it("sends nothing when a field is missing", async () => {
    const puts = renderForm()
    await chooseSeries("Matahari & Bulan")

    fill("Code", "SM-P")
    save()

    await screen.findByText("Name is required.")
    expect(puts).toEqual([])
  })

  it("sends nothing for a code the API would refuse", async () => {
    const puts = renderForm()
    await chooseSeries("Matahari & Bulan")

    fill("Code", "SM P/1")
    fill("Name", "Promo")
    fill("Release date", "2019-05-08")
    save()

    await screen.findByText(/Code may contain only/)
    expect(puts).toEqual([])
  })

  it("shows the API's refusal inline", async () => {
    renderForm(() =>
      json(
        {
          status: 422,
          title: "Unprocessable Entity",
          detail: "series not found",
        },
        422
      )
    )
    await chooseSeries("Matahari & Bulan")

    fill("Code", "SM-P")
    fill("Name", "Promo")
    fill("Release date", "2019-05-08")
    save()

    const alert = await screen.findByRole("alert")
    expect(alert.textContent).toContain("series not found")
    expect(screen.queryByText("Saved SM-P.")).toBeNull()
  })

  it("shows a duplicate code as the API's conflict, not as an update", async () => {
    const writes = renderForm(() =>
      json(
        {
          status: 409,
          title: "Conflict",
          detail: "an expansion set with this code already exists",
        },
        409
      )
    )
    await chooseSeries("Matahari & Bulan")

    fill("Code", "SM1")
    fill("Name", "Promo")
    fill("Release date", "2019-05-08")
    save()

    const alert = await screen.findByRole("alert")
    expect(alert.textContent).toContain("already exists")
    expect(writes.map((w) => w.method)).toEqual(["POST"])
  })
})
