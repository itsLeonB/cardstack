// Shared plumbing for the lists that scroll on the API's `page` / `limit` /
// `meta.total` contract: catalog search, Collection entries, Master Inventory.

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
  data: { detail?: string; data?: unknown }
}

export type ListPage<T> = OkPage<T> | FailedPage

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
        throw new Error(response.data.detail ?? fallbackMessage)
      }
      return response
    },
  }
}
