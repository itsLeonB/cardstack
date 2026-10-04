// Shared plumbing for the lists that scroll on the API's `page` / `limit` /
// `meta.total` contract: catalog search, Collection entries, Master Inventory.
import type { CardSummary } from "@/generated/models"

interface PageMeta {
  page: number
  limit: number
  total: number
}

interface OkPage<T> {
  status: 200
  data: { data?: T[] | null; meta: PageMeta }
}

// `data` lists both `detail` and `data` so every generated response variant is
// assignable (same trick as `StatusResponse` in `lib/collections.ts`).
type FailedPage = {
  status: number
  data: { code?: string; detail?: string; data?: unknown }
}

export type ListPage<T> = OkPage<T> | FailedPage

/** What `InfiniteCardResults` renders: the loaded cards and the server's total. */
export interface CardList {
  cards: CardSummary[]
  total: number
}

/**
 * The API's 401 for what a Guest may not do (ADR-0013's `login_required`
 * code). A distinct class so the UI shows a sign-in prompt, not a generic error.
 */
export class LoginRequiredError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "LoginRequiredError"
  }
}

const isOk = <T>(page: ListPage<T>): page is OkPage<T> => page.status === 200

/**
 * Next 1-indexed page, or undefined at the end. A short page also ends the
 * list: `total` and the rows come from separate statements, so it can be stale.
 */
export function nextPageParam(last: ListPage<unknown>) {
  if (!isOk(last)) return undefined
  const { page, limit, total } = last.data.meta
  const loaded = last.data.data?.length ?? 0
  return loaded === limit && page * limit < total ? page + 1 : undefined
}

/**
 * Flatten pages into one list, dropping repeats by id: a row inserted ahead of
 * the window between two fetches repeats the boundary row.
 */
export function mergePages<T>(pages: ListPage<T>[], idOf: (row: T) => string) {
  const byId = new Map<string, T>()
  let total = 0
  for (const page of pages) {
    if (!isOk(page)) continue
    total = page.data.meta.total
    for (const row of page.data.data ?? []) {
      const id = idOf(row)
      if (!byId.has(id)) byId.set(id, row)
    }
  }
  return { rows: [...byId.values()], total }
}

/**
 * The query options every page-scrolling list shares. The generated fetcher
 * resolves for every HTTP status; throw so an error body is never stored as a
 * page (`nextPageParam` would read `meta` off it).
 */
export function infinitePages<P extends FailedPage>(
  fetchPage: (page: number | undefined, signal: AbortSignal) => Promise<P>,
  fallbackMessage: string
) {
  return {
    initialPageParam: 1,
    getNextPageParam: nextPageParam,
    queryFn: async ({
      pageParam,
      signal,
    }: {
      pageParam: number | undefined
      signal: AbortSignal
    }) => {
      const response = await fetchPage(pageParam, signal)
      if (response.status !== 200) {
        const message = response.data.detail ?? fallbackMessage
        throw response.status === 401 && response.data.code === "login_required"
          ? new LoginRequiredError(message)
          : new Error(message)
      }
      return response
    },
  }
}
