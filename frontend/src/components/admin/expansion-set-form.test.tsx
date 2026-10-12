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
  ungroupedExpansionSets: [],
}

interface Put {
  path: string
  body: unknown
}

/** Stubs the API: GET /catalog/series and PUT /admin/expansion-sets/{code}. */
function renderForm(putResponse: () => Response = () => savedResponse()) {
  const puts: Put[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = new URL(String(input)).pathname
      if (init?.method === "PUT") {
        puts.push({ path, body: JSON.parse(String(init.body)) })
        return putResponse()
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
  return puts
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

function savedResponse() {
  return json({ data: { id: "x", code: "x", name: "x", imageUrl: "" } })
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
        path: "/admin/expansion-sets/SM-P",
        body: {
          name: "Kartu Promo Matahari & Bulan",
          releaseDate: "2019-05-08",
          seriesId: "series-sm",
        },
      },
    ])
  })

  it("edits an existing set: fills its fields, keeps its code, sends the changes", async () => {
    const puts = renderForm()

    await screen.findByRole("option", { name: /SM1/ })
    fireEvent.change(screen.getByLabelText("Existing set"), {
      target: { value: "SM1" },
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
        path: "/admin/expansion-sets/SM1",
        body: {
          name: "Koleksi Matahari (diperbaiki)",
          releaseDate: "2016-12-02",
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
})
