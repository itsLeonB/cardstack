import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useMatches,
} from "@tanstack/react-router"
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools"
import { TanStackDevtools } from "@tanstack/react-devtools"
import type { QueryClient } from "@tanstack/react-query"

import { AppShell } from "@/components/layout/app-shell"
import { NotFound } from "@/components/layout/not-found"
import { ThemeProvider } from "@/components/theme-provider"
import appCss from "../styles.css?url"

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  head: () => ({
    meta: [
      {
        charSet: "utf-8",
      },
      {
        name: "viewport",
        content: "width=device-width, initial-scale=1",
      },
      {
        title: "Cardstack",
      },
      {
        name: "description",
        content:
          "Track every card you own, across every binder. Browse the Pokémon TCG catalog and organize your Collections.",
      },
      {
        name: "theme-color",
        content: "#fbbf24",
        media: "(prefers-color-scheme: light)",
      },
      {
        name: "theme-color",
        content: "#1c1c14",
        media: "(prefers-color-scheme: dark)",
      },
    ],
    links: [
      {
        rel: "icon",
        href: "/favicon.svg",
        type: "image/svg+xml",
      },
      {
        rel: "manifest",
        href: "/manifest.json",
      },
      {
        rel: "stylesheet",
        href: appCss,
      },
    ],
  }),
  notFoundComponent: NotFound,
  component: RootLayout,
  shellComponent: RootDocument,
})

// `/auth` brings its own minimal shell (routes/auth/route.tsx), so the main
// shell steps aside for it.
function RootLayout() {
  const isAuthRoute = useMatches({
    select: (matches) => matches.some((match) => match.routeId === "/auth"),
  })
  if (isAuthRoute) return <Outlet />
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  )
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <HeadContent />
      </head>
      <body>
        <ThemeProvider>{children}</ThemeProvider>
        <TanStackDevtools
          config={{
            position: "bottom-right",
          }}
          plugins={[
            {
              name: "Tanstack Router",
              render: <TanStackRouterDevtoolsPanel />,
            },
          ]}
        />
        <Scripts />
      </body>
    </html>
  )
}
