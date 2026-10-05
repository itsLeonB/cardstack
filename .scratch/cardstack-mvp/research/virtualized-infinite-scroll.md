# Offset pagination, infinite scroll and TanStack Virtual for the card lists

Research date: 2026-10-02. Scope: ticket 25 (`issues/25-research-virtualized-infinite-scroll.md`). How the catalog search and the Collection detail card lists could load more cards on scroll with the existing `page` / `limit` / `meta.total` API, whether list virtualization is worth it, and what it does to filters in the URL, offset drift, the +/- quantity batching (tickets 20, 22, 23, 29), facets (ticket 21), accessibility and tests.

Every claim is followed by its source. "Prototype result" means I ran it myself in a throwaway project outside the repo (`/tmp/claude-1000/-home-leon-Projects-itsLeonB-cardstack/1499eae9-6979-4183-9138-65ffa95ff661/scratchpad/proto`, not part of the repo; no file under `frontend/` or `backend/` was touched). Prototype stack: `@tanstack/react-query` 5.103.1 and `@tanstack/react-router` 1.170.41 (the same versions installed in `frontend/node_modules`), `@tanstack/react-virtual` 3.14.13 (current `latest` on npm as of today; it depends on `@tanstack/virtual-core` 3.17.11), React 19.3.0, vitest 4 + jsdom 30 for logic tests, and Playwright 1.56.1 driving headless Chromium (build 1194) for browser tests. The browser prototype measurements ran on the dev box with no CPU throttling, so treat absolute milliseconds as indicative only; the ratios are what matter. Where something was not verified I say "Not confirmed".

## TL;DR

- Recommendation: replace the numbered pager with `useInfiniteQuery` over the existing `page`/`limit`/`total` contract, rendered as a plain DOM grid with a "Load more" button that doubles as an IntersectionObserver sentinel, dedupe by card id, and no `page` in the URL. Backend needs no change (the ordering is already a total order ending in `cards.id`).
- Do not add TanStack Virtual yet: at realistic depths (a few hundred tiles) plain DOM costs ~11k elements / 7 MB per 500 tiles (prototype), and `content-visibility: auto` cuts the relayout cost 12-19x without losing find-in-page, tab order or focus. Virtualization is a separate, gated ticket; the prototype shows the row-chunked grid works.
- The real hazards are not in the virtualizer: `useQuantityBatch.prune()` shows stale quantities once pages append (reproduced), the catalog quantities lookup caps at `limit` 100, `gcTime: 0` on the Collection entries query defeats scroll restoration on back navigation, and refetch/focus refetch re-requests every loaded page sequentially.
- Proposed split: 4 follow-up tickets (M, M, S, S) plus one optional, evidence-gated L for virtualization; see section 8.

## 1. Combining `useInfiniteQuery` with `page` / `limit` / `total`, and detecting "near the bottom"

### 1.1 Contract today

**Confirmed.** Catalog search takes `page` (1-indexed, min 1, default 1) and `limit` (default 24, max 100) and returns `{ data, meta: { total, page, limit } }`. Source: `backend/internal/adapters/http/handler/catalog_handler.go:68-69` (struct tags `default:"1" minimum:"1"` and `default:"24" minimum:"1" maximum:"100"`); `backend/internal/domain/dto/pagination.go` (`PaginationMeta{Total, Page, Limit}`); `frontend/src/generated/endpoints/catalog/catalog.zod.ts` (`SearchCatalogCardsResponse.meta` has exactly `limit`, `page`, `total`). The service clamps `limit` to `maxCardSearchLimit` (100) and `page` to at least 1: `backend/internal/domain/service/catalog_service.go` (`normalizePagination`).

**Confirmed.** The Collection entries list and the Master Inventory list use the same machinery: both call `catalog.SearchCards` with `CollectionID` or `ProfileID` set and return the same `PaginationMeta`. Source: `backend/internal/domain/service/inventory_service.go` (`List`, `ListMasterInventory`), `backend/internal/domain/repository/catalog_repository.go` (`SearchCards`, `cardsBase`). Collection entries accept `limit` up to 100 as well (`backend/internal/adapters/http/handler/inventory_handler.go:64,80`).

**Confirmed.** `total` and the page rows come from two separate statements (`Count`, then `Find`) with no explicit transaction, so `total` can disagree with the rows actually returned if data changes in between. Source: `catalog_repository.go` `SearchCards` (`base.Count(&total)` then `base...Limit().Offset().Find()`), and `GetGormInstance` only joins a transaction when the ctx carries one (`docs/agents/conventions/backend.md`, "Layout").

### 1.2 `getNextPageParam` from `total`

**Confirmed.** The infinite-query docs describe `getNextPageParam(lastPage, allPages, lastPageParam, allPageParams)`, `hasNextPage` being true when it returns something other than `null`/`undefined`, and the "API without a cursor" recipe that derives the next param from `lastPageParam`. Source: TanStack Query docs, Infinite Queries guide (`docs/framework/react/guides/infinite-queries.md`, sections "Example" and "What if my API doesn't return a cursor?").

Recommended shape (prototype result, `logic.test.ts` a1-a4): `getNextPageParam: (last) => last.data.length === last.meta.limit && last.meta.page * last.meta.limit < last.meta.total ? last.meta.page + 1 : undefined`, with `initialPageParam: 1`. The `data.length === limit` clause is the guard for the count/page race above: a short page ends pagination even if `total` is stale. Prototype results: 25 rows at limit 10 gives 3 pages and `hasNextPage` false with a 5-row last page; `total` 30 issues exactly requests 1, 2, 3 and no fourth; an empty result issues one request and `hasNextPage` is false.

**Confirmed.** The generated client's fetchers resolve for every HTTP status (they return a `{status, data}` union rather than throwing), and `data` is typed nullable. Source: `frontend/src/generated/endpoints/catalog/catalog.ts` (`searchCatalogCards` returns the `customFetch` result), `catalog.zod.ts` (`"data": ...nullable()`), and the existing call sites that branch on `data.status === 200`. Consequence: the infinite `queryFn` must throw (or map) on a non-200 response, otherwise an error body is stored as a page and `getNextPageParam` reads `meta` off it. Not prototyped; derived from reading the code.

### 1.3 orval can generate the infinite hooks, with a distinct key

**Confirmed.** orval supports `override.query.useInfinite: true` and `useInfiniteQueryParam: 'page'` (a string or candidate list); only operations that declare that query param get an infinite hook. Source: orval configuration reference (`docs/content/docs/reference/configuration/output.mdx`, sections `useInfinite`, `useInfiniteQueryParam`).

Prototype result (orval 8.33.0 run against `backend/openapi.json` in a scratch directory, output not in the repo): it generates `getSearchCatalogCardsInfiniteQueryOptions`, `useSearchCatalogCardsInfinite` and `getSearchCatalogCardsInfiniteQueryKey`, and the same for `listCollectionEntries` and `listMasterInventory`. The infinite key is `['infinite', '/catalog/cards', params]`, the plain key is `['/catalog/cards', params]`, so the two shapes cannot collide in the cache. The generated `queryFn` injects `pageParam` as `page` (`{...params, 'page': pageParam ?? params?.page}`) and the caller must still supply `initialPageParam` and `getNextPageParam` in the options. Two practical consequences: (a) any existing prefix invalidation such as `invalidateQueries({ queryKey: getListCollectionEntriesQueryKey(id) })` will not match an infinite query, so invalidations must also use the `Infinite` key helper; (b) the scratch run printed "Could not determine the installed @tanstack/react-query version, so hooks are generated for v4", so the real config needs `override.query.version: 5` (or `output.packageJson`) set when adopting this. Not confirmed: whether the repo's `baseUrl` runtime option changes the key prefix (the existing plain key includes `import.meta.env.VITE_API_BASE_URL`; my scratch config had no baseUrl).

### 1.4 `maxPages` and its trade-offs

**Confirmed (docs).** `maxPages` keeps only the last N pages in the cache, needs `getPreviousPageParam` to fetch backwards, and exists to bound memory and the cost of refetching "dozens of pages". Source: Infinite Queries guide, "What if I want to limit the number of pages?".

Prototype result (`logic.test.ts` b5-b7, limit 10, `maxPages: 3`): after 7 fetches the cache holds `pageParams [5,6,7]` and the first retained row is `Card 00040`, so row offsets must be derived from `pageParam` (`(page-1)*limit`) instead of array index; `hasPreviousPage` is true; a refetch costs 3 requests (5, 6, 7); `fetchPreviousPage` re-adds page 4 and drops page 7. Trade-offs: the rows above the window vanish (the document gets shorter and the user's scroll position jumps unless the UI compensates), scroll restoration cannot return to a dropped page, `aria-posinset` needs the page offset, and an "up" fetch path must exist. Recommendation: do not use `maxPages` for the first version; only reconsider it together with virtualization if sessions routinely exceed ~80 pages.

### 1.5 What happens when many pages are refetched

**Confirmed (docs and source).** When an infinite query goes stale and is refetched, "each group is fetched sequentially, starting from the first one"; if the results are ever removed from the cache, pagination restarts at the first page. Source: Infinite Queries guide, "What happens when an infinite query needs to be refetched?"; implementation in `@tanstack/query-core/src/infiniteQueryBehavior.ts` (the `do ... while (currentPage < remainingPages)` loop with `getNextPageParam` computed from the previous result; it `break`s when that returns `null`).

Prototype results (`logic.test.ts`, node, fake 5 ms-20 ms API):
- b1: with 6 pages loaded, `invalidateQueries` issued requests 1,2,3,4,5,6, strictly sequential (each started after the previous ended); wall time scaled with page count (121 ms at 20 ms per request).
- b2/b3: window focus refetch (needs `QueryClient.mount()`, which `QueryClientProvider` does in an app) re-requested all 5 loaded pages at `staleTime: 0`, and 0 requests at `staleTime: 60_000`. This matches the docs: refetch on focus only applies to stale queries (Query docs, Window Focus Refetching guide).
- b4: if the total shrinks while 5 pages are loaded (50 rows to 22), the refetch stops early because `getNextPageParam` returns undefined: 3 pages, 22 rows. So a refetch can shorten the list.
- c2: after A (3 pages) to B to A, the cached 3 pages of A appear immediately and are then refetched sequentially (requests 1, 2, 3).
- h1: calling `fetchNextPage()` while a background refetch of 4 pages is in flight cancels that refetch with the default `cancelRefetch: true` (requests 1,2,5; two aborted signals; pages 3-4 keep stale content and page 5 is appended); with `{ cancelRefetch: false }` the call is ignored and the refetch completes (requests 1,2,3,4). This is what the docs warn about ("calling `fetchNextPage` while an ongoing fetch is in progress runs the risk of overwriting data refreshes", Infinite Queries guide). Rule for the implementation: trigger `fetchNextPage` only when `!isFetching`, not just `!isFetchingNextPage` (the official Virtual example guards only `isFetchingNextPage`, see 1.6), and pass `cancelRefetch: false` on the button.
- a5: two `fetchNextPage()` calls back to back with the default option issue the page request twice (both completed in my fake API); with `cancelRefetch: false` one request.

Implication for the app: catalog search currently uses default focus refetching with `staleTime: 0`, and the Collection entries query refetches on focus unless edits are pending (`collection-entries.tsx`, `refetchOnWindowFocus: () => !batch.isBusy()`). With N loaded pages each tab refocus costs N sequential requests. Mitigations, cheapest first: a `staleTime` of tens of seconds on the infinite catalog query (catalog data changes only on ingestion; the 0 default is unnecessary there); on the Collection page keep the focus refetch (ticket 29 wants it) but accept N requests, which is small if the page size is raised (100 is the maximum, section 2.6).

### 1.6 Detecting "scrolled near the bottom" with TanStack Virtual

**Confirmed.** The official infinite-scroll example sets `count` to `hasNextPage ? allRows.length + 1 : allRows.length` (a loader row), reads the last virtual item with `getVirtualItems()`, and calls `fetchNextPage()` in an effect when `lastItem.index >= allRows.length - 1 && hasNextPage && !isFetchingNextPage`, with `overscan: 5`. Source: TanStack Virtual repo, `examples/react/infinite-scroll/src/main.tsx`.

Prototype result (browser, `/virt`): the same idea applied to grid rows (fetch when the last rendered row index is within 3 rows of the loaded row count) loaded pages without double requests: over 8 scroll-to-bottom steps the cache went 3, 5, 7 ... 17 pages with `fetchNextPage` called exactly `pages - 1` times. Overscan controls how early the trigger fires (it extends the rendered range, so the last virtual item is `overscan` rows ahead of what is visible); `overscan: 2` plus the 3-row threshold meant a fetch started well before the bottom. Source for overscan semantics: Virtual docs, API `overscan` ("The number of items to render above and below the visible area") and `defaultRangeExtractor` in `@tanstack/virtual-core/src/index.ts` (`start = max(startIndex - overscan, 0)`, `end = min(endIndex + overscan, count - 1)`).

Without a virtualizer (the recommended baseline) the same trigger is an IntersectionObserver on the "Load more" button with a generous `rootMargin`. Prototype result (`/feed`, `rootMargin: 800px`): scrolling to 8 pages issued `fetchNextPage` 7 times with no duplicate requests.

## 2. Virtualizing a responsive grid of card tiles

### 2.1 What the grid and the tile look like today

**Confirmed.** Results render as `ul.grid` with `grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4` (viewport breakpoints) inside `PageContainer variant="wide"` (`max-w-6xl`, i.e. 1152 px). Each tile is a `Card` with a lazy `<img loading="lazy">` (`aspect-[5/7] w-full`), a name link, a set link, rarity and category badges, optional tag badges, and an optional control slot (the `QuantityControl`: two buttons with SVG icons and a number input). Source: `frontend/src/components/catalog/card-results.tsx`, `card-tile.tsx`, `components/collections/quantity-control.tsx`, `components/layout/page-container.tsx`. The app shell uses window scrolling with a `sticky top-0 z-40` header: `components/layout/app-shell.tsx`, `site-header.tsx:39`.

**Confirmed.** The data sizes: the catalog holds 12,119 distinct cards (`issues/04-unified-catalog-ingestion.md`, manual verification section); a Collection is bounded by the user's own data and an optional `maxCardCount` (hundreds at most per the ticket; `GLOSSARY.md` Collection definition).

### 2.2 The pattern: virtualize rows, chunk items into columns

**Confirmed.** Virtual's own `lanes` option is documented as "the number of lanes the list is divided into (aka columns for vertical lists...). Items are assigned to the lane with the shortest total size", with `laneAssignmentMode` `'estimate'` (default) or `'measured'`, i.e. a masonry layout. Source: Virtual docs, API reference `lanes` / `laneAssignmentMode`. With variable tile heights (tag badges wrap) lanes produce independently-stacked columns rather than aligned rows, which differs from the current CSS grid (a row is as tall as its tallest tile). Not prototyped: I did not build the lanes variant; the conclusion is from the documented assignment rule.

Therefore the pattern for an aligned grid is a row virtualizer: compute `columns` from the container width, chunk the flat item list into rows of `columns` items, virtualize the rows with `measureElement` for variable row height, and render each row as its own CSS grid. Prototype result (`/virt`): this renders ~20-35 tiles (about 470-815 DOM elements) at any depth, with no overlapping or gapped rows measured after resizes.

### 2.3 Measuring columns, and what happens when they change on resize

Prototype implementation: a `ResizeObserver` on the list container sets `width`; `columns = clamp(floor((width + gap) / (minTile + gap)), 2, 5)`. This replaces the viewport-based Tailwind breakpoints with container-based ones (an unavoidable design change: `grid-cols-*` classes can no longer decide the column count because the JS needs the same number to chunk rows). A `matchMedia` mirror of the Tailwind breakpoints is the alternative that keeps the exact current breakpoints; not prototyped.

Prototype result (`e2`, scrolled to ~5000 px with 6 pages, viewport 1280 to 900 to 700 to 400 to 1280):
- Naive (just re-chunk on the new column count): no overlaps or gaps, but the visible card drifts because the same `scrollY` maps to different cards (top card `c00050` at 5 columns, `c00040` at 4, `c00030` at 3, `c00020` at 2): the user loses their place on every resize or device rotation.
- Fix: remember the index of the first card in the first visible row on each render with stable columns; on a column change call `virtualizer.measure()` (drops cached row heights, which are keyed by row index by default and no longer describe the same content) and `scrollToIndex(floor(anchor / columns))`. Result: top card `c00048` held at 4, 3 and 2 columns (within one row), `c00045` back at 5 columns.
- Gotcha found: calling `scrollToIndex` synchronously inside the layout effect that handles the column change landed ~15 rows away when the document got shorter (widening 2 to 5 columns: `c00120` instead of `c00048`, reproducible in two runs). Deferring the call by one `requestAnimationFrame` fixed it (`c00045`). Cause not diagnosed (the document height shrinks in the same commit); treat re-anchoring as needing an e2e test.
- Row heights (measured with the default `measureElement`) differ from `estimateSize` (460 px estimate vs. about 464 px between measured row tops), so `getTotalSize()` drifts by a few px per row as rows are measured; this is why restored positions were off by up to ~36 px (section 3.3).

### 2.4 Window scroller vs. element scroller, sticky header

**Confirmed.** `useWindowVirtualizer` exists for window scrolling, and `scrollMargin` is "the space between the beginning of the scrolling element and the start of the list", to be subtracted in the item's `translateY` (`item.start - virtualizer.options.scrollMargin`); it can be measured with `getBoundingClientRect()` or a ResizeObserver "in scenarios when items above your virtual list might change their height". Source: Virtual docs API `scrollMargin`; the `examples/react/window/src/main.tsx` example. `scrollPaddingStart` / `scrollPaddingEnd` options exist in the installed `virtual-core` (`src/index.ts:359-360`) and apply to `scrollToIndex`.

Recommendation: the window scroller. The app shell scrolls the window and `useElementScrollRestoration` / the router's own restoration already track `window` (section 3). An element scroller would need a fixed-height inner container, a second scrollbar, and `data-scroll-restoration-id`. Prototype result (`/virt`, window virtualizer, 56 px sticky header, a 220 px filter block above the list): `scrollMargin` was measured from the list's `getBoundingClientRect().top + scrollY` in a ResizeObserver (the filter panel height can change when chips appear), and `scrollPaddingStart: 56` keeps `scrollToIndex` targets from hiding under the sticky header (top card landed at y=56 after the re-anchor).

### 2.5 Is virtualization needed at these sizes?

Prototype result (production build, headless Chromium, replica tile with the same element structure as `CardTile` + `QuantityControl`, about 22 DOM elements per tile; the real tile without the control is fewer; images are same-origin SVGs so no real network cost):

| Rendered tiles | Elements | JS heap | Forced relayout on a width change | Initial render to first paint | Image requests with `loading="lazy"` |
|---|---|---|---|---|---|
| plain 500 | 11,143 | 7.1 MB | 13-27 ms | 226-296 ms | 120 |
| plain 2,000 | 44,518 | 20.8 MB | 59-67 ms | 491-639 ms | 120 |
| plain 12,000 | 267,018 | 111.7 MB | 421-499 ms | 2.6-3.0 s | 120 |
| virtualized, any of 500 / 2,000 / 12,000 loaded | 472 (20 tiles) to ~815 (35 tiles) | 3.8-5.2 MB | 1.3-1.7 ms | n/a (data seeded) | 45-55 |
| plain 2,000 + `content-visibility: auto` | 44,518 | 20.8 MB | 4.9 ms | 292 ms | 120 |
| plain 12,000 + `content-visibility: auto` | 267,018 | 111.7 MB | 25.9 ms | 1.4 s | 120 |

Reading: `loading="lazy"` already bounds image requests (120 at a 1280x800 viewport regardless of N; MDN: lazy "defers loading the image until it reaches a calculated distance from the viewport"). The costs that grow with N are DOM nodes, heap and style/layout. Honest conclusion:
- Up to a few hundred tiles (a Collection, or a filtered catalog) plain DOM is adequate: 500 tiles is 7 MB and a ~13-27 ms relayout.
- Around 2,000 tiles (about 83 pages of 24) plain DOM starts to cost (relayout ~60 ms), but `content-visibility: auto` with `contain-intrinsic-size` cuts layout work 12x (58.7 to 4.9 ms) while keeping all nodes findable (`window.find` still found the last card: prototype `e7`). MDN lists `content-visibility` as Baseline 2024 ("Newly available", since September 2024; developer.mozilla.org/en-US/docs/Web/CSS/Reference/Properties/content-visibility).
- Only a user who scrolls the unfiltered catalog toward its 12k cards needs real virtualization, and nothing in the tickets says anyone should. Not confirmed on real low-end phones: I did not CPU-throttle or test mobile Safari; a 4-6x slower CPU would move each threshold down proportionally.
- Numbering pages as today (24 tiles, ~530 elements) is cheapest of all; its drawback is UX, not performance.

Image `loading="lazy"` note (MDN, `<img>` reference): lazy images need reserved space; the real tile uses `aspect-[5/7] w-full`, which reserves it, so no layout shift is expected from lazy loading (not separately tested).

### 2.6 Page size

For infinite scroll prefer a larger `limit` than 24: fewer requests, fewer page boundaries (fewer offset-drift opportunities, and cheaper focus refetch). 60 is a multiple of 2, 3, 4 and 5 columns, 100 is the API maximum. A Collection of up to 100 entries loads in a single request at `limit=100`. These are design suggestions, not tested.

## 3. Interaction with filters in URL search params

### 3.1 How the routes handle `page` today

**Confirmed.** `catalogSearchSchema` is derived from the generated zod params minus `limit`, so it carries `page` (int, min 1, default 1) plus the four multi-value filters (`frontend/src/lib/catalog-search.ts`). The catalog search route uses it via `validateSearch`, derives `loaderDeps` from the filters and `page` (dropping `collectionId`), prefetches the cards with `ensureQueryData(getSearchCatalogCardsQueryOptions(deps))`, and resets `page: 1` on every filter change while page changes push a new history entry (`frontend/src/routes/catalog/search.tsx`). The Collection detail route reuses the same schema with no loader prefetch for entries (`routes/_authenticated/collections/$collectionId/index.tsx`), and the Expansion Set browse route has its own `{ page }` schema (`routes/catalog/sets/$expansionSetId.tsx`). Master Inventory also renders through `CardResults` (`components/inventory/master-inventory.tsx`). Router config already has `scrollRestoration: true` (`frontend/src/router.tsx:14`).

### 3.2 Reset on filter change, loaderDeps, query keys

**Confirmed (docs).** `loaderDeps` "defines a cache key during route planning", the deps are compared with deep equality, and when they change the route reloads regardless of `staleTime`; the default `gcTime` for loader data is 5 minutes. Source: TanStack Router docs, Data Loading guide ("Using `loaderDeps` to access search params", "Key options", "Some Important Defaults").

Design: put the filters, and nothing else, in both `loaderDeps` and the infinite query key; drop `page` from the URL. Prototype results:
- c1/c2 (node): a different filter value is a different cache entry, the first filter's pages come back instantly on return and are then refetched in full (see 1.5).
- e5 (browser): navigating to a new filter value (a new history entry) loads page 1 of the new key and the router scrolls to the top (`scrollY` 3000 to 0, 35 tiles to 20); `Back` restores the previous list at `scrollY` 3000 with 35 tiles from the cache. So "reset on filter change" needs no explicit code beyond the key and the navigation.
- l1/l2 (node): in the loader, `queryClient.infiniteQuery({ ...options, staleTime: 'static' })` reuses cached pages (3 loader calls, 1 API request), whereas `staleTime: 0` re-requested every call. The installed `@tanstack/query-core` marks `ensureInfiniteQueryData` as deprecated in favor of exactly this (JSDoc in `src/queryClient.ts`, "Use queryClient.infiniteQuery({ ...options, staleTime: 'static' }) instead"). The existing loader's `ensureQueryData` pattern therefore ports with that call. Because the loader only awaits the first page, the route renders as soon as page 1 is ready.

### 3.3 Scroll restoration on back navigation

**Confirmed (docs and source).** Router scroll restoration is keyed by `location.state.__TSR_key` (one entry per history entry; falls back to `href`), snapshots scroll positions of `window` and any scrolled elements before navigation and persists them to `sessionStorage` on `pagehide`, then calls `scrollTo` once after the destination has rendered (`onRendered`). Source: Router docs, Scroll Restoration guide ("Custom Cache Keys"); `@tanstack/router-core/src/scroll-restoration.ts` (`defaultGetScrollRestorationKey`, `snapshotCurrentScrollTargets`, the `onRendered` handler that calls `scrollTo({ top: scrollY })`). The same guide names virtualized lists as the case needing manual handling, with `useElementScrollRestoration({ getElement: () => window })` feeding the virtualizer's `initialOffset`; Virtual's docs add `takeSnapshot()` / `initialMeasurementsCache` for restoring measured sizes (API reference, `initialMeasurementsCache`).

Because the restore is a single `scrollTo` right after render, it only works if the document is already tall enough at that moment, i.e. only if the cached pages render synchronously. Prototype results (browser, leave to a card detail page, `Back`):

| Variant | Position at leaving | After Back (+1.5 to 2.6 s) |
|---|---|---|
| Plain DOM list, default `gcTime` (`/feed`) | `scrollY` 3000, 8 pages | `scrollY` 3000, same card at the same offset, 8 pages from cache |
| Virtualized, default `gcTime`, router restoration only | `scrollY` 4000, card c40 at 36 px | `scrollY` 3964, c40 at 84 px (off by ~36 px, estimated vs. measured row heights) |
| Virtualized, default `gcTime`, plus `useElementScrollRestoration` `initialOffset` | `scrollY` 4000 | `scrollY` 4000 exactly, but the top card is one row different (c35 vs c40) for the same reason |
| `gcTime: 0` (plain or virtualized) | `scrollY` 3000-4000, 6-8 pages | `scrollY` 0, only page 1-2 in the cache: the restore `scrollTo` is clamped by a document that is only one page tall, and the router does not retry |
| Virtualized, `gcTime: 0` plus `initialOffset` | `scrollY` 3988 | `scrollY` 2107 (clamped), 3 pages |

**Confirmed finding that affects ticket 22.** The Collection entries query is deliberately `gcTime: 0` with `refetchOnMount: "always"` so a card taken to 0 does not reappear stale (`components/collections/collection-entries.tsx`; ticket 22 acceptance criteria). With an infinite query that setting means back navigation from a card detail page lands at the top with one page. Options: (a) accept it on the Collection page (it is a bounded list; with `limit=100` most Collections are one page, so there is nothing to restore); (b) use the default `gcTime` with `refetchOnMount: "always"` so the cached pages show immediately and are then refetched, accepting that tiles at 0 disappear after that refetch (the same trade-off ticket 29 already accepted for focus refetch). I recommend (a) for the first version and revisiting only if Collections regularly exceed one page.

**Not confirmed:** whether these results hold in the real app. The prototype is a client-only Vite single-page app; the app itself is TanStack Start with `setupRouterSsrQueryIntegration` (`frontend/src/router.tsx`), and in dev a direct load is server-rendered (`docs/agents/conventions/frontend.md`, last entry), so loader and restoration behaviour could differ; verify in the real app with Playwright. Also not confirmed: restoration after a hard reload (the router persists to `sessionStorage` on `pagehide`, but the query cache is gone so only page 1 would exist, the same clamp); restoration inside iOS Safari's bfcache.

### 3.4 Linkable state and the `page` param

Options for `page` in the URL: (1) drop it (recommended); (2) keep it as the start page (`initialPageParam: page`) so a shared link opens deep in the list, which cannot scroll upward without `getPreviousPageParam`, `fetchPreviousPage` and an upward trigger, i.e. the bi-directional list from the Infinite Queries guide, and it makes `aria-posinset` and the "N of total" status offset-based; (3) mirror the loaded count into the URL with `replace` (rejected: churn for little value).

Recommendation: drop `page` from `catalogSearchSchema` (`.omit({ page: true })`) and from the Expansion Set route schema; the filter set stays linkable, which is what the ticket calls linkable state. Old `?page=3` links should degrade to page 1: unknown search keys are stripped by a zod object schema by default, but I did not verify that against the repo's router, so add a test (Not confirmed). Call sites that pass `page: 1` in `Link search` (e.g. the set page's "Search within this set" link) need updating; `bun run typecheck` will find them.

## 4. Offset pagination pitfalls

### 4.1 Is the ordering stable? Confirmed yes

**Confirmed.** `SearchCards` orders by `expansion_sets.release_date DESC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC`. The last key is the primary key, so the order is total and deterministic. Source: `backend/internal/domain/repository/catalog_repository.go:368`. The Collection entries and Master Inventory lists use the same function (section 1.1), so they share it. The other ORDER BY clauses in the repo (`inventory_repository.go:73`, `ListHoldings`) are not paginated lists. No backend change is needed.

PostgreSQL's documentation is the reason this matters: "When using LIMIT, it is important to use an ORDER BY clause that constrains the result rows into a unique order. Otherwise you will get an unpredictable subset of the query's rows... using different LIMIT/OFFSET values to select different subsets of a query result will give inconsistent results unless you enforce a predictable result ordering with ORDER BY." and "The rows skipped by an OFFSET clause still have to be computed inside the server; therefore a large OFFSET might be inefficient." Source: PostgreSQL docs, 7.6 LIMIT and OFFSET (postgresql.org/docs/current/queries-limit.html). The second quote is also the reason deep scrolling gets slower per request; not measured here (no database run).

### 4.2 Drift when data changes between page fetches

Even with a total order, offset pages drift when rows are inserted or removed ahead of the current offset. Prototype results (`logic.test.ts` f1-f4, limit 10):
- f1: a row inserted ahead of the loaded window between fetching page 2 and page 3 makes page 3 start one row early, so the last card of page 2 is duplicated (`Card 00019` appeared twice). Client-side dedupe by `id` (a `Map` keyed on id when flattening) fixes the visible duplicate (30 rows to 29 unique).
- f2: a row removed ahead of the window between fetches makes the next page skip one row (`Card 00020` was never shown). Dedupe cannot fix a skip.
- f3: a full refetch of all loaded pages (invalidate) repairs both, because it re-reads from page 1 (the docs claim this too: "we're not using stale cursors and potentially getting duplicates or skipping records", Infinite Queries guide). It is not atomic: pages are fetched sequentially, so drift during the refetch itself is still possible (Not confirmed experimentally).

Where this actually happens in the app:
- The catalog only changes on ingestion, so drift there is mostly theoretical.
- The Collection entries list is the live case. Taking a card to 0 deletes the entry on the server (ticket 20: quantity 0 removes) but ticket 22 keeps the tile on screen at 0, so the server list is now one row shorter than the loaded client list: the next page skips the card that would have followed (f2 scenario). Adding a card from the catalog in another tab inserts into the sorted order, which duplicates a boundary card (f1). So in one session on the Collection page: dedupe is required, and an unhealed skip is possible until the next refetch.
- Mitigations in increasing cost: dedupe by id (required, trivial); larger page size so fewer boundaries (section 2.6); on the Collection page, when one or more cards were saved at 0 since the last full load, make "Load more" do `refetch()` first (all loaded pages, which drops the zero rows) and then `fetchNextPage` (a small amount of code; not prototyped); keyset pagination (rejected below).

### 4.3 Cursor / keyset pagination: rejected for now

Out of scope per the ticket and not needed: the ordering key is a 5-column composite (`release_date` nullable, then set id, local id, name, card id), so a keyset cursor would need a composite, null-aware comparison and a new API contract on three endpoints (catalog, Collection entries, Master Inventory) plus regenerated clients. It would remove drift and the O(offset) cost, but it also removes random access to a page (the `page` URL param, any later "jump to"), which the numbered pager used. Revisit only if drift on Collection pages proves a real problem or OFFSET cost shows up in measurements.

## 5. Interaction with +/- quantity controls, pending edits and facets

### 5.1 `useQuantityBatch.prune()` breaks when pages append (reproduced)

**Confirmed (prototype result, `prune.test.tsx`, using the repo's `use-quantity-batch.ts` copied verbatim with only the import paths rewritten and a fake bulk endpoint).** Both `CollectionEntries` and `CollectionCardResults` run `useEffect(() => batch.prune(), [query.data])` (`collection-entries.tsx`, `collection-card-results.tsx`). `prune()` deletes the optimistic override for every card with nothing outstanding. With `useQuery` the data only changes on a refetch, which is fresh. With `useInfiniteQuery` every `fetchNextPage` produces a new `data` object while pages 1..N still hold their old rows, and a successful save does not touch the cache (`onSaved` on the Collection page invalidates master inventory and counts, not entries; ticket 22 requires the tile to stay at 0). Result of the scenario (30 entries at quantity 3, pages of 10):
- p1/p2: set a card on page 1 to 5 and save (server 5, displayed 5); then `fetchNextPage()`: displayed 3, the stale pre-edit value, while the server has 5.
- p3/p4: set a card to 0 and save (displayed 0, entry removed on the server); after the next `fetchNextPage()` it displays 3 again although the server no longer has it.

This is the most important implementation hazard in the whole design, and it is independent of virtualization.

**Fix candidate, verified in the prototype (p5).** Write the confirmed quantity into every cached row after a save, with `queryClient.setQueryData` over `pages[]`, so that the cache is the source of truth and `prune()` becomes harmless; a card saved at 0 stays in the cache at quantity 0 (so the tile stays, per ticket 22) and is dropped by the next real refetch. With that patch the same scenario showed 5 and 0 after `fetchNextPage()`. Requirements: `useQuantityBatch` must pass the bulk response's per-item results to `onSaved` (today it calls `onSaved?.()` with no arguments, `lib/use-quantity-batch.ts`), and the same patch is needed for each cache that renders a quantity: the Collection entries pages, and the catalog's quantity lookups (5.3). Alternative: prune only on a full refetch (not on a page append), which is simpler but leaves the cache stale after a save.

### 5.2 `setQueryData` across `pages[]` and invalidation cost

Prototype result (g1): patching one card inside a 5-page cache (`pages.map(...)` replacing only the page that contains the card) took 0.89 ms and preserved the identity of the other four pages (`true,true,false,true,true`), so memoised per-page rows do not re-render. Shape rule from the docs: keep `{ pages, pageParams }` intact ("Make sure to always keep the same data structure of pages and pageParams!", Infinite Queries guide, "What if I want to manually update the infinite query?"). Invalidation, by contrast, costs one sequential request per loaded page (1.5), so do not invalidate the infinite entries query after each save: patch instead. The existing `invalidateCollectionCounts` and `invalidateMasterInventory` calls are unaffected, but note that orval's infinite keys start with `'infinite'` (1.3), so a Master Inventory list that moved to an infinite query would no longer be matched by `invalidateMasterInventory`'s prefix keys.

### 5.3 The catalog's quantity lookup does not scale with accumulated pages

**Confirmed (code reading).** `CollectionCardResults` requests `useListCollectionEntries(collectionId, { cardId: cardIds, limit: cardIds.length })` for the cards on the current page (`collection-card-results.tsx`), the API caps `limit` at 100, and the ids go into the query string. With accumulated pages `cardIds.length` exceeds 100 after a few pages and the query string grows with every page, so it breaks. It needs one lookup per loaded page (for example `useQueries`, each keyed by that page's ids, with `limit` equal to the page size, at most 100). Each of those caches has the same stale-after-save problem as 5.1, so the same `setQueryData` patch (or per-page invalidation) is needed. Not prototyped; derived from the code and the API limit.

### 5.4 Window focus refetch, pending edits, and ticket 29

**Confirmed.** Ticket 29 already makes the Collection entries query refetch on focus only when `!batch.isBusy()`. With infinite pages, that refetch re-requests every loaded page (1.5, b2), replaces the cache in one swap at the end (all pages land together, no partial list; from reading `infiniteQueryBehavior.ts`, where `fetchFn` returns the accumulated result once), drops rows at quantity 0, and may shorten the list. That is acceptable under ticket 29's accepted trade-off, but it should be guarded further: `refetchOnWindowFocus` already returns false while edits are pending, and `fetchNextPage` must additionally never run while `isFetching` (h1), or a focus refetch would be cancelled by the scroll trigger. The catalog search query has no such guard and should get a `staleTime` (1.5).

### 5.5 Facets are not tied to pagination

**Confirmed.** The search response metadata carries only `total`, `page`, `limit`; facets come from separate endpoints (`/catalog/facets`, `/collections/{id}/facets`, `/inventory/cards/facets`) whose params are the filters without `page`/`limit`. Source: `catalog.zod.ts` (`SearchCatalogCardsResponse.meta`; `ListCatalogFacetsQueryParams` has no `page`/`limit`), `frontend/src/lib/catalog-search.ts` `toFacetParams` (drops `page`), and the facets hooks in `collection-entries.tsx` and `routes/catalog/search.tsx` (`placeholderData: keepPreviousData`). So adding infinite scroll changes nothing for ticket 21: the facets query key does not contain the page and is not refetched when a page is appended. The only coupling is on the catalog search page when a Collection is selected (ticket 23): `CollectionCardResults.onSaved` invalidates that Collection's entries and facets after a save (`collection-card-results.tsx`), using the plain `getListCollectionEntriesQueryKey` / `getListCollectionFacetsQueryKey` prefixes. That keeps working as long as the quantity lookups stay plain queries (5.3); if the Collection entries move to an infinite query, their invalidations must use the `Infinite` key helpers because the keys start with `'infinite'` (1.3).

### 5.6 What a tile unmounting does to a pending edit

Pending edits survive because their state lives in `useQuantityBatch` (refs plus `quantities`/`errors` state in the parent), not in the tile. What is lost when a tile unmounts (only with virtualization): the `QuantityControl`'s typed draft text (`useState<string | null>` inside the tile) and the visible `role="alert"` message if it is mounted only while the tile is rendered (the error text itself stays in `errors`, so it reappears when the tile re-mounts). Plain DOM avoids both.

## 6. Accessibility

### 6.1 Keyboard and focus in virtualized lists

Prototype results (browser, `/virt`, production-like markup):
- Focus loss: with focus on a tile's "Increase quantity" button, scrolling far away unmounts that tile and `document.activeElement` becomes `BODY`. The user loses their place.
- Find-in-page: with 72 cards loaded, `window.find` (Chromium's find API, used as a stand-in for Ctrl+F) found a rendered card but returned false for a loaded-but-unrendered one; on the plain 500-tile list it found the last card. Chromium's built-in find bar behaves the same way for text that is not in the DOM (Not confirmed with the real find bar UI).
- Keyboard, virtualized with a virtual-index trigger and no button (`/virt`): from the last rendered tile, 150 Tab presses never reached the footer; the page kept loading more (6 to 8 pages) because tabbing scrolls the focused element into view and the trigger fires. WCAG 2.1.1 (Keyboard) requires all functionality to be keyboard operable, and 2.4.1 Bypass Blocks concerns skipping repeated blocks; neither SC names infinite scroll, so I do not claim a failure, only that in that variant the footer is unreachable while more content loads. The current footer holds a wordmark, a brand line and descriptive text (`components/layout/site-footer.tsx`), so the practical impact today is low; it matters if footer links are added. Sources: WCAG 2.2 Understanding SC 2.1.1 and 2.4.1 (w3.org/WAI/WCAG22/Understanding/keyboard.html, bypass-blocks.html).
- Tab order: unmounted tiles are not in the order at all, by construction.
- Prototype result for the recommended plain-DOM baseline with a focusable "Load more" button after the list (`/feed`, test `e9`): from the last tile's "Increase" button, the first Tab lands on the button (which also triggers the next page load, 2 to 3 pages), and the second Tab reaches the footer link; the new tiles are inserted before the button in DOM order, so focus is not dragged down the list. So with a focusable button the footer is reachable by keyboard. With the `disabled` attribute during fetching, the first Tab skipped the button and went straight to the footer (also reachable, but the button is not operable at that moment). For wheel or touch users the footer never enters the viewport while pages remain (after 6 scroll-to-bottom steps: 9 pages loaded, footer not in view); that is inherent to infinite scroll and is acceptable only because the footer has no links today. A plain list with an invisible, non-focusable sentinel and no button would behave like the `/virt` case for keyboard users (reasoning from the e3 result, not separately tested). Pressing Enter on the focused "Load more" button loaded the next page. The button must not use the `disabled` attribute while fetching: with `disabled={isFetching}` the focused button lost focus to `BODY` after Enter; with `aria-disabled` and a guard in the handler focus stayed on the button (`BUTTON#more`). The repo's shadcn `Button` supports `disabled`, so this is an easy mistake.

### 6.2 Roles and set size: `list` / `listitem` vs. `feed`

**Confirmed.** The WAI-ARIA Authoring Practices "Feed Pattern" defines `role=feed` as "a section of a page that automatically loads new sections of content as the user scrolls", whose items are `article` elements, each with `aria-labelledby`, `aria-posinset` and `aria-setsize` ("or -1 if the total is undetermined"), `aria-busy` on the feed while multiple DOM operations run, and recommends keyboard commands (Page Down / Page Up between articles, Ctrl+End / Ctrl+Home to leave the feed) that it notes have no established convention. It also says the feed establishes a contract where the page loads/removes articles based on which article has focus. Source: APG, Feed Pattern (w3.org/WAI/ARIA/apg/patterns/feed/).

Recommendation: do not adopt `role=feed`. Reasons: it implies automatic loading plus custom key handling that this app would have to build; the tiles are not articles; and the repo's e2e specs and conventions count results by `listitem` (`docs/agents/conventions/frontend.md`, several entries, and `e2e/catalog-search.spec.ts`), which `article` would silently break. The baseline keeps `ul` / `li` and adds `aria-posinset` / `aria-setsize` only if a virtualizer is introduced.

If virtualization is added, markup matters. Prototype result with axe (`@axe-core/playwright`, tags wcag2a/2aa/21aa/22aa/best-practice): a `div[role=list]` containing row `div`s that contain `ul > li` produced one `aria-required-children` violation; `div[role=list]` > row `div[role=presentation]` > `div[role=presentation]` grid > `div[role=listitem]` with `aria-posinset` / `aria-setsize` produced none (the two remaining findings, `html-has-lang` and `page-has-heading-one`, are artifacts of my bare prototype page). Note `ul > div` directly is also invalid HTML, so rows cannot sit inside the existing `ul`.

### 6.3 Status announcements, "Load more", reduced motion

- Status messages: WCAG 4.1.3 requires status messages to be programmatically determinable "through role or properties such that they can be presented... without receiving focus"; the techniques it lists are ARIA22 (`role=status`) and ARIA23 (`role=log`). Source: WCAG 2.2 Understanding SC 4.1.3 (w3.org/WAI/WCAG22/Understanding/status-messages.html). The current results use `aria-live="polite"` on the "N cards" line (`card-results.tsx`); for infinite scroll keep one polite status region reading "N of total cards loaded" (a polite region is enough; do not announce every page).
- Explicit "Load more": it is the keyboard/assistive equivalent for scroll-triggered loading (WCAG 2.1.1) and, being focusable and placed after the list, keeps the footer reachable by keyboard (prototype `e9`, 6.1). Make the same button the IntersectionObserver sentinel so the auto-load and the manual path share one element (prototype `/feed`).
- Reduced motion: nothing in this design needs animation. Programmatic scrolls (`scrollToIndex` for column re-anchoring, router restoration) should use instant behavior; the router already exposes `scrollRestorationBehavior` (`@tanstack/router-core/src/router.ts:500`) and Virtual's `scrollToIndex` accepts a `behavior` option. MDN: `prefers-reduced-motion` exists to avoid "scaling or panning large objects" that "can trigger discomfort for those with vestibular motion disorders" (developer.mozilla.org, `@media/prefers-reduced-motion`); WCAG 2.3.3 Animation from Interactions is AAA (Understanding page). Skeleton shimmer for the "loading more" row, if added, should honor the media query.
- Screen readers: not tested with a real screen reader (no NVDA, VoiceOver or TalkBack in this environment); everything above comes from the specs and automated checks. Not confirmed.

## 7. Test strategy

### 7.1 jsdom (Vitest, the repo already uses `environment: "jsdom"` in `frontend/vite.config.ts`)

- Pure logic and cache behaviour need no DOM: the `getNextPageParam`, dedupe, `maxPages`-free page window, and the `setQueryData` patch can be unit-tested with a bare `QueryClient` and a fake API (all of `logic.test.ts` ran in plain Node). Hook tests with `renderHook` + `QueryClientProvider` covered the prune regression (`prune.test.tsx`).
- Baseline (no virtualizer) component tests: render the infinite results with a mocked fetcher, click "Load more", assert tiles and the status text. No layout needed.
- IntersectionObserver and ResizeObserver do not exist in jsdom (prototype `j5`: both `undefined`). For the sentinel approach, stub a minimal `IntersectionObserver` class that records its callback and `observe`d elements, and fire the callback inside `act()` to simulate intersection (same style as the repo's `src/test-setup.ts` global config). TanStack Virtual guards a missing `ResizeObserver` itself (`virtual-core/src/index.ts:118, 513`), so it does not throw in jsdom.
- Virtualizer in jsdom (prototype `j1-j6`, installed virtual-core 3.17.11): an element virtualizer renders 0 rows by default because jsdom reports a 0x0 scroller (j1) and `initialRect` alone does not help because the measured rect overrides it (j2); stubbing `observeElementRect` with a fixed rect (the technique Virtual's own tests use, `packages/react-virtual/tests/index.test.tsx`: `observeElementRect: (_, cb) => cb({ height, width })` and `measureElement: () => itemSize`) renders 4 rows at 400 px / 100 px estimate (j3); `useWindowVirtualizer` renders using jsdom's `innerHeight` of 768 (9 rows, j4). The default `measureElement` reads `entry.borderBoxSize`, then a cached size, then `offsetHeight` (`virtual-core/src/index.ts:247-290`, not `getBoundingClientRect`, despite the docs prose), so mock `HTMLElement.prototype.offsetHeight` (j6: total size moved from 10,000 px to 10,750 px once rows measured 250) or pass a custom `measureElement`. Conclusion: jsdom can check the wiring (row chunking by a stubbed column count, trigger calls `fetchNextPage` when the last row index is near the end), not real layout.

### 7.2 Playwright e2e (the repo has `frontend/e2e/*.spec.ts`)

Needs a real browser: scroll-triggered loading (IntersectionObserver), filter change resetting to the top, back-navigation scroll restoration (cache and `gcTime` interplay, section 3.3), column changes on resize and re-anchoring (the `scrollToIndex` timing bug in 2.3 only shows in a browser), focus retention on the "Load more" button, and an axe pass with `@axe-core/playwright` (already a devDependency). Constraints from the conventions: stubs of paginated lists must include `meta: { total, page, limit }` and echo the request `origin` (`docs/agents/conventions/frontend.md`); the specs count results with `getByRole("listitem")`, which keeps working for plain DOM (rendered count grows after each load) and for `div[role=listitem]` tiles under virtualization, but with a virtualizer a `toHaveCount(n)` only sees rendered tiles. Existing seeded e2e data is small, so all tiles render. Not confirmed: whether the e2e seed has more than one page of cards for the scroll test; a stubbed API (`page.route`) would give control either way.

## 8. Recommendation

### 8.1 Design

1. Data: `useInfiniteQuery` over the existing contract (orval `useInfinite: true`, `useInfiniteQueryParam: "page"`, `version: 5`), `initialPageParam: 1`, `getNextPageParam` from `total` plus the short-page guard, `queryFn` throws on non-200, items flattened and deduplicated by `id`, no `maxPages`. Page size 60 on the catalog, 100 on Collection entries and Master Inventory. Loader: `queryClient.infiniteQuery({ ...options, staleTime: "static" })` for page 1.
2. URL: filters only; remove `page` from the schemas and from `loaderDeps`. Filter change is a new history entry (router resets scroll; the new query key resets pages). Default `gcTime` on the catalog so back navigation restores (plain DOM restored exactly in the prototype). Collection entries keep `gcTime: 0` for now and accept landing at the top.
3. UI: the existing grid and tile unchanged (plain DOM), plus one polite status line "N of total cards loaded", and a "Load more" button that is also the IntersectionObserver sentinel (`rootMargin` about 800 px, guarded by `!isFetching`, `fetchNextPage({ cancelRefetch: false })`, `aria-disabled` not `disabled`). Optional `content-visibility: auto; contain-intrinsic-size: auto <row height>` on tiles once lists regularly pass ~1,000 tiles.
4. Quantities: pass bulk results to `onSaved` and `setQueryData`-patch confirmed quantities across `pages[]` (Collection) and per-page lookups (catalog), so `prune()` is harmless; dedupe by id; add a `staleTime` to the infinite catalog query; keep ticket 29's focus guard and add `!isFetching` to the scroll trigger.
5. Backend: no change.

### 8.2 Trade-offs

- Wins: the smallest change (the grid, tile, `QuantityControl`, facets and filter panel stay as they are); tab order, find-in-page, focus and text selection keep working; e2e `listitem` counts keep working; scroll restoration works through the router's defaults.
- Costs: DOM grows with every page (about 22 elements per tile with the quantity control); the footer is pushed away while more pages exist; a long unfiltered session of thousands of tiles will feel heavy without `content-visibility` and heavier without virtualization; deep pages cost O(offset) in SQL (not measured); the Collection page can show one skipped card after a card was zeroed (4.2) until a refetch; the catalog's per-page quantity lookups multiply with pages.

### 8.3 Rejected or deferred alternatives

- Virtualization now (TanStack Virtual `useWindowVirtualizer` + row chunking): works (section 2) but costs focus loss, find-in-page, a required ARIA markup change, restoration quirks, a column-change re-anchor bug class, test complexity and a change from viewport to container breakpoints, for a benefit that only appears past roughly 2,000 rendered tiles. Deferred to a gated ticket.
- Keeping numbered pages as they are: zero risk and cheapest DOM, but it does not meet the ticket's goal of loading on scroll.
- Plain IntersectionObserver sentinel with no button: simplest to code but there is no keyboard/AT-operable control and, by reasoning from the `/virt` keyboard result (e3, a virtual-index trigger; a plain-DOM invisible sentinel was not separately tested), keyboard users would tab through new tiles instead of reaching the footer; rejected in favor of the button-as-sentinel.
- Cursor / keyset pagination: out of scope, composite null-aware key across three endpoints, loses random access; see 4.3.
- `maxPages`: see 1.4.
- `role=feed`: see 6.2.
- A sparse per-page `useQuery` model (virtual rows each mapped to `page = floor(index / limit) + 1`, total known up front) would make restoration and focus refetch cheap because only visible pages are active, and removes the all-pages refetch cascade; it needs custom page bookkeeping and was not prototyped, so it is only a candidate for the gated virtualization ticket.

### 8.4 Proposed follow-up tickets (dependency order)

1. (M) Infinite catalog results: orval infinite hooks and key helpers, the shared infinite data hook (flatten, dedupe, `getNextPageParam`, throw on non-200), `CardResults` "Load more" + sentinel + status line + `aria-disabled`, drop `page` from catalog search and Expansion Set routes and `loaderDeps`, loader with `infiniteQuery({staleTime:'static'})`, catalog `staleTime`, update `Link search` call sites, unit tests (IntersectionObserver stub) and an e2e for scroll/filter reset/back restoration. Blocked by: nothing.
2. (M) Collection entries and Master Inventory on infinite: entries infinite query (page size 100), `useQuantityBatch` passes results to `onSaved`, `setQueryData` patch across `pages[]`, `prune()` regression test (the p1-p5 scenario), `!isFetching` guard for the trigger, ticket 29 focus rules, switch every prefix invalidation that must reach these caches (`invalidateMasterInventory`, and any `getListCollectionEntriesQueryKey(id)` invalidation that targets an entries list that became infinite) to the orval `Infinite` key helpers because infinite keys start with `'infinite'` (1.3, 5.2), `gcTime` decision recorded, the zeroed-card skip mitigation if wanted. Blocked by: 1.
3. (S) Catalog quantities lookup per loaded page (replace `{ cardId: all ids, limit: ids.length }` with one `useQueries` entry per page, each at most 100 ids) with the same save-time patch; keeps ticket 23 behaviour. Blocked by: 1 and the `onSaved`-results change from 2.
4. (S) Measure on real devices before deciding on virtualization: CPU-throttled and a real phone on a production build with a seeded ~2,000-tile scroll, and decide on `content-visibility: auto` (S, one CSS rule) versus the next ticket. Blocked by: 1.
5. (L, optional, only if ticket 4 shows a need) Virtualized grid: `useWindowVirtualizer` with row chunking, container-width columns, `scrollMargin` and `scrollPaddingStart`, re-anchor on column change (deferred `scrollToIndex`), `role=list` / presentational rows / `div[role=listitem]` with `aria-posinset` and `aria-setsize`, focus handling, restoration via `useElementScrollRestoration` + `initialOffset`, jsdom stubs and Playwright coverage. Blocked by: 1, 2, 3, 4.

Rough sizes: S is under half a day, M about one to two days, L three to five days including e2e. These are estimates, not measured.

## Sources

- TanStack Query docs, Infinite Queries guide: https://github.com/TanStack/query/blob/main/docs/framework/react/guides/infinite-queries.md (read raw from the repo)
- TanStack Query docs, Window Focus Refetching and Query Invalidation guides: same repo, `docs/framework/react/guides/window-focus-refetching.md`, `query-invalidation.md`
- `@tanstack/query-core` 5.103.1 source in `frontend/node_modules`: `src/infiniteQueryBehavior.ts`, `src/queryClient.ts` (`infiniteQuery`, deprecated `ensureInfiniteQueryData`), `src/removable.ts` (default `gcTime` 5 minutes)
- TanStack Virtual docs: https://tanstack.com/virtual/latest (introduction, API virtualizer reference incl. `lanes`, `scrollMargin`, `initialOffset`, `initialMeasurementsCache`, React adapter); examples `examples/react/infinite-scroll` and `examples/react/window` in https://github.com/TanStack/virtual; tests `packages/react-virtual/tests/index.test.tsx`
- `@tanstack/virtual-core` 3.17.11 and `@tanstack/react-virtual` 3.14.13 source (installed in the scratch project; npm registry `latest` as of 2026-10-02)
- TanStack Router docs: Scroll Restoration, Data Loading (`loaderDeps`) guides: https://github.com/TanStack/router/tree/main/docs/router/guide (`scroll-restoration.md`, `data-loading.md`); `@tanstack/router-core` 1.170.x source in `frontend/node_modules` (`src/scroll-restoration.ts`, `src/router.ts`), `@tanstack/react-router` `src/ScrollRestoration.tsx`
- orval configuration reference (`useInfinite`, `useInfiniteQueryParam`): https://github.com/orval-labs/orval, `docs/content/docs/reference/configuration/output.mdx`; orval 8.33.0 generator output (scratch run)
- PostgreSQL docs, LIMIT and OFFSET: https://www.postgresql.org/docs/current/queries-limit.html
- W3C WAI-ARIA Authoring Practices, Feed Pattern: https://www.w3.org/WAI/ARIA/apg/patterns/feed/
- WCAG 2.2 Understanding documents: 2.1.1 Keyboard, 2.4.1 Bypass Blocks, 2.4.3 Focus Order, 2.3.3 Animation from Interactions, 4.1.3 Status Messages (https://www.w3.org/WAI/WCAG22/Understanding/)
- MDN: `content-visibility`, `prefers-reduced-motion`, `<img loading>` (developer.mozilla.org)
- Repo files read: `frontend/src/routes/catalog/search.tsx`, `routes/catalog/sets/$expansionSetId.tsx`, `routes/_authenticated/collections/$collectionId/index.tsx`, `components/catalog/{card-results,card-tile,collection-card-results}.tsx`, `components/collections/{collection-entries,quantity-control}.tsx`, `components/layout/{app-shell,site-header,site-footer,page-container}.tsx`, `lib/{catalog-search,collections,master-inventory,use-quantity-batch}.ts`, `router.tsx`, `orval.config.ts`, `src/generated/endpoints/**`, `e2e/catalog-search.spec.ts`; `backend/internal/domain/repository/{catalog,inventory}_repository.go`, `domain/service/{catalog,inventory}_service.go`, `adapters/http/handler/{catalog,inventory}_handler.go`, `domain/dto/pagination.go`; `docs/agents/conventions/{general,frontend,backend}.md`; tickets 04, 20, 21, 22, 23, 25, 29
- Prototype (throwaway, outside the repo): `/tmp/claude-1000/-home-leon-Projects-itsLeonB-cardstack/1499eae9-6979-4183-9138-65ffa95ff661/scratchpad/proto` (`src/logic.test.ts`, `src/prune.test.tsx`, `src/jsdom.test.tsx`, `src/loader.test.ts`, `src/app/main.tsx`, `pw-exp.mjs`, `axe.mjs`) and `.../scratchpad/orval-try`
