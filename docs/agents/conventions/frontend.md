# Frontend code conventions

These apply to every frontend change, whether made by the root agent or a subagent. Read `general.md` in this folder as well. A rule here holds across modules. How one module works lives in that file's header comment, and a decision with rejected alternatives lives in an ADR (see `general.md`).

## General

- Extract a component once reuse is real, not in anticipation of it.
- Treat everything in the client bundle as public: render no unsanitized HTML and ship no secrets.
- When you couldn't verify a UI change in a browser, say so instead of claiming success.
- Treat `src/generated/**` (orval) and `src/routeTree.gen.ts` as build outputs. Change the source (the OpenAPI spec, the route files) and regenerate: `bun run codegen` for orval, and the TanStack Start Vite plugin rebuilds the route tree. orval `override.operations` keys are the spec's operationIds (`"search-catalog-cards"`), not the generated function names; a wrong key silently generates nothing.
- `.oxlintrc.json` is the source of truth for lint rules, including the `anti-slop/*` set. `bun run lint` must pass.
- Format proactively: run `bun run format` after editing so `bun run check` (Prettier) passes. `tools/oxlint/anti-slop` comes from an externally installed plugin, so it stays in `.prettierignore` and is never reformatted by hand.

## Routing and layout

- On public routes, read auth state with `useSession()` (no redirect) and gate features on `isAuthenticated`; only `requireAuth` under `_authenticated/` redirects. Guards read Clerk through `context.auth`, and a layout with such a guard sets `ssr: false`.
- Every redirect target must pass `isSameOriginPath` (`lib/route-guard.ts`) and is followed with `router.history.push`, since `navigate({ to })` doesn't take a query string.
- Drop user-scoped query data in `createSessionChangeHandler` (`lib/session.ts`), which reacts to Clerk's session changes, not in a mutation's `onSuccess`. Never store the Clerk session token.
- Lay out every page with `PageContainer` and `PageHeader` (`components/layout/`). Pages never render their own `main`, wrapper `div` or `h1`: each shell owns its single `main` landmark.
- Every route sets its document title with `head: () => pageHead("<Name>")` (`lib/site.ts`); detail routes use the entity name from their `loader`. Indexing and Open Graph decisions are in ADR-0014.
- Name test files under `src/routes/` `-<name>.test.tsx`: the router plugin treats the `-` prefix as "not a route" and warns on any other file that doesn't export a `Route`.

## UI

- Read and set the theme through `ThemeProvider` / `useTheme`; don't touch the `dark` class or `localStorage` directly.
- Use `text-muted-foreground` only where contrast is verified, and never `text-primary` (amber) for text on light surfaces. Style inline links with `text-foreground` and an underline.
- Make a whole card the click target with the stretched-link pattern: the title is the one `Link` whose `after:absolute after:inset-0` pseudo-element covers the `relative` card, and secondary actions sit above it with `relative z-10`. Don't nest interactive elements inside the link or wrap the card in an anchor.
- Show a toast (`toast.error` from `sonner`) for failures of non-form actions such as logout. Keep form validation, dialog errors, per-card edits and failed page data inline (`role="alert"`), with a Retry action on failed page data. Mutations resolve for every HTTP status, so a toast needs both the non-success `status` branch in `onSuccess` and `onError`.
- Render nav links, breadcrumbs and filter chips as plain links or buttons, never `ul`/`li`: e2e specs count results by `listitem`.

## Data and caching

- Debounce with TanStack Pacer's `useDebouncer` (`@tanstack/react-pacer`), not hand-rolled `setTimeout`/`clearTimeout` timers; pass `onUnmount: (d) => d.flush()` when pending work must survive unmount, and call the latest handler through a ref to avoid stale closures.
- Edit quantities through `useQuantityBatch` + `QuantityControl`, and after a save patch the cached infinite list (`patchCollectionEntryQuantities`) instead of refetching it.
- Build a long list on the shared infinite-list stack (`lib/infinite-pages.ts`, `lib/infinite-*.ts`, `components/catalog/virtual-grid.tsx`) and read those files' header comments first. The URL carries filters only, never `page`.
- Infinite query keys start with `'infinite'`, so the plain key helper misses them. To invalidate a list, call `invalidateMasterInventory`, `invalidateCollectionEntries` or `invalidateCollectionCounts` (`lib/collections.ts`), and call `invalidateCollectionCounts` next to `invalidateMasterInventory` on every Inventory Entry change. Never invalidate with a bare key helper (a cache patch targets the infinite key on purpose).
- A `select` result must be a plain object or array, never a `Map` or `Set`: structural sharing only preserves identity for plain values, and an effect keyed on `query.data` loops forever on a fresh `Map` each render.

## Testing

- Test an invalidation with real keys in a real `QueryClient` (`isInvalidated`), as `master-inventory.test.ts` and `collections.test.tsx` do, because a mocked `invalidateQueries` can't notice a prefix that misses.
- Virtualized lists render only a window of `listitem`s, so Playwright checks on them assert through the status line ("N of total cards loaded") or a request log, not `listitem` counts. Unit tests of virtualized components call `stubGridLayout()` (`src/test-grid-layout.ts`).
- Fake network functions in component tests of infinite lists must take a tick (`await new Promise((r) => setTimeout(r, 1))`), because the grid asks for the next page only after it renders a fetch as in flight. Playwright against the dev server can see a page requested twice (StrictMode remount), so assert the set of requested pages, not their counts.
- Playwright stubs of paginated API lists must include `meta: { total, page, limit }`, and stubbed CORS responses echo the request's `origin` header instead of hard-coding a port.
- Playwright on virtualized lists: click an in-viewport link when asserting scroll restoration, `locator.focus()` is not necessarily `:focus-visible`, and load every page first to test reaching the end by keyboard.
- Playwright checks of a layout's child routes reach them by client navigation, because the dev server renders a direct load on the server, where browser-level API stubs don't apply.
- Playwright specs that need a session sign in through `e2e/support/clerk-auth.ts` against the real Clerk development instance, never a stub of Clerk. Never `fill()` a credential. Put the describe under `{ tag: SIGNED_IN_TAG }` with `useSignedInSuite()`. Register API stubs before signing in. Why, and how it runs: `docs/agents/testing.md`.
