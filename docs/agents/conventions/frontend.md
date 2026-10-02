# Frontend code conventions

These apply to every frontend change, whether made by the root agent or a subagent. Read `general.md` in this folder as well.

- Extract a component once reuse is real, not in anticipation of it.
- Treat everything in the client bundle as public: render no unsanitized HTML and ship no secrets.
- When you couldn't verify a UI change in a browser, say so instead of claiming success.
- Treat `src/generated/**` (orval) and `src/routeTree.gen.ts` as build outputs. Change the source (the OpenAPI spec, the route files) and regenerate: `bun run codegen` for orval, and the TanStack Start Vite plugin rebuilds the route tree.
- Colocate unit tests beside the source as `*.test.ts(x)`. End-to-end Playwright tests live in `e2e/`.
- `.oxlintrc.json` is the source of truth for lint rules, including the `anti-slop/*` set. `bun run lint` must pass.
- On public routes, read auth state with `useSession()` (no redirect) and gate features on `isAuthenticated`; only `requireAuth` under `_authenticated/` redirects.
- Quantity editing goes through `useQuantityBatch` + `QuantityControl`; new surfaces reuse them rather than re-implementing batching, ordering or decline handling.
- The `cardId` entries filter treats an empty list as "no restriction": disable the query when there are no ids rather than sending an empty list.
- Debounce with TanStack Pacer's `useDebouncer` (`@tanstack/react-pacer`), not hand-rolled `setTimeout`/`clearTimeout` timers; pass `onUnmount: (d) => d.flush()` when pending work must survive unmount, and call the latest handler through a ref to avoid stale closures.
- Multi-value filter dropdowns use `MultiSelect` (`components/ui/multi-select.tsx`, a base-ui Popover of native checkboxes); the popover only mounts when open, so tests and e2e must click the trigger first. Render active-selection chips as buttons, not `li`, because e2e specs count results by `listitem`.
- Read and set the theme through `ThemeProvider` / `useTheme` (`components/theme-provider.tsx`); don't touch the `dark` class or `localStorage` directly.
- Header and footer nav links come from `useNavLinks` (`components/layout/nav-links.ts`); add new top-level destinations there. Keep nav as plain links in a `nav`, with no list markup, because e2e specs count `listitem`.
- Pages must not render their own `<main>`: `AppShell` provides the single `main` landmark, so use a `<div>`.
- Use `text-muted-foreground` only where contrast is verified; axe flags low-contrast shell text.
- Lay out every page with `PageContainer` and `PageHeader` (`components/layout/`); pages never render their own wrapper `div`, `main` or `h1`. `PageContainer` variants: `narrow` for auth pages, `default` for everything else, `wide` only for catalog search and Master Inventory. `PageHeader` renders the page's single `h1`, with an optional `description` and `actions` slot.
- Nested pages show `Breadcrumbs` (`components/layout/breadcrumbs.tsx`) above the header: ancestors are links, the last crumb is the current page. Top-level pages (Catalog, Collections, Master Inventory, Account) have none, and no page renders a "Back to ..." link. Breadcrumbs are plain links in a `nav`, with no list markup, because e2e specs count `listitem`.
