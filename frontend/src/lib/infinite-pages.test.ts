import { describe, expect, it } from "vitest"
import {
  LoginRequiredError,
  infinitePages,
  mergePages,
  nextPageParam,
} from "./infinite-pages"
import type { ListPage } from "./infinite-pages"

type Row = { id: string }
type Page = ListPage<Row>

const row = (id: string): Row => ({ id })
const idOf = (r: Row) => r.id

function ok(
  ids: string[],
  meta: { page: number; limit: number; total: number }
): Page {
  return { status: 200, data: { data: ids.map(row), meta } }
}

const failed: Page = { status: 500, data: { detail: "boom" } }

describe("nextPageParam", () => {
  it("returns the next page while rows remain", () => {
    expect(nextPageParam(ok(["a", "b"], { page: 1, limit: 2, total: 5 }))).toBe(
      2
    )
  })

  it("stops once total is reached", () => {
    expect(
      nextPageParam(ok(["a", "b"], { page: 3, limit: 2, total: 6 }))
    ).toBeUndefined()
  })

  it("stops on a short page even if total says more", () => {
    expect(
      nextPageParam(ok(["a"], { page: 1, limit: 2, total: 9 }))
    ).toBeUndefined()
  })

  it("stops on an error response", () => {
    expect(nextPageParam(failed)).toBeUndefined()
  })
})

describe("mergePages", () => {
  it("flattens pages in order and reports the latest total", () => {
    const merged = mergePages(
      [
        ok(["a", "b"], { page: 1, limit: 2, total: 4 }),
        ok(["c", "d"], { page: 2, limit: 2, total: 5 }),
      ],
      idOf
    )
    expect(merged.rows.map(idOf)).toEqual(["a", "b", "c", "d"])
    expect(merged.total).toBe(5)
  })

  it("drops a row repeated across a page boundary", () => {
    const merged = mergePages(
      [
        ok(["a", "b"], { page: 1, limit: 2, total: 4 }),
        ok(["b", "c"], { page: 2, limit: 2, total: 4 }),
      ],
      idOf
    )
    expect(merged.rows.map(idOf)).toEqual(["a", "b", "c"])
  })

  it("ignores non-200 pages and null rows", () => {
    const empty: Page = {
      status: 200,
      data: { data: null, meta: { page: 1, limit: 2, total: 0 } },
    }
    expect(mergePages([failed, empty], idOf)).toEqual({ rows: [], total: 0 })
  })
})

describe("infinitePages", () => {
  const signal = new AbortController().signal

  it("fetches the requested page, starting at 1", async () => {
    const page = ok(["a"], { page: 2, limit: 1, total: 3 })
    const fetchPage = (n: number | undefined) => {
      expect(n).toBe(2)
      return Promise.resolve(page)
    }
    const options = infinitePages(fetchPage, "fallback")
    expect(options.initialPageParam).toBe(1)
    expect(await options.queryFn({ pageParam: 2, signal })).toBe(page)
  })

  it("throws the error detail, or the fallback, so an error body is never stored as a page", async () => {
    await expect(
      infinitePages(() => Promise.resolve(failed), "fallback").queryFn({
        pageParam: 1,
        signal,
      })
    ).rejects.toThrow("boom")
    await expect(
      infinitePages(
        () => Promise.resolve<Page>({ status: 500, data: {} }),
        "fallback"
      ).queryFn({ pageParam: 1, signal })
    ).rejects.toThrow("fallback")
  })
  it("throws LoginRequiredError for a 401 login_required, so a prompt can replace the generic error", async () => {
    const locked: Page = {
      status: 401,
      data: { detail: "sign in to use this part of the catalog" },
    }
    // `code` is the stable signal; the detail text is never parsed.
    const withCode = {
      ...locked,
      data: { ...locked.data, code: "login_required" },
    }
    await expect(
      infinitePages(() => Promise.resolve(withCode), "fallback").queryFn({
        pageParam: 1,
        signal,
      })
    ).rejects.toBeInstanceOf(LoginRequiredError)
  })

  it("keeps a 401 without the login_required code a plain error", async () => {
    const expired = () =>
      infinitePages(
        () =>
          Promise.resolve<Page>({ status: 401, data: { detail: "expired" } }),
        "fallback"
      ).queryFn({ pageParam: 1, signal })
    await expect(expired()).rejects.toThrow("expired")
    await expect(expired()).rejects.not.toBeInstanceOf(LoginRequiredError)
  })
})
