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
import { Toaster } from "@/components/ui/sonner"
import { SITE_NAME, SITE_URL } from "@/lib/site"
import appCss from "../styles.css?url"

const DESCRIPTION =
  "Track every card you own, across every binder. Browse the Pokémon TCG catalog and organize your Collections."

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
      { name: "description", content: DESCRIPTION },
      // Static on purpose: link-preview crawlers don't run scripts, so every
      // URL shares these (SPA mode; per-route Open Graph isn't attempted).
      { property: "og:type", content: "website" },
      { property: "og:site_name", content: SITE_NAME },
      { property: "og:title", content: "Cardstack: track every Pokémon card you own" },
      { property: "og:description", content: DESCRIPTION },
      { name: "twitter:card", content: SITE_URL ? "summary_large_image" : "summary" },
      ...(SITE_URL
        ? [
            { property: "og:image", content: `${SITE_URL}/og-image.png` },
            { property: "og:image:width", content: "1200" },
            { property: "og:image:height", content: "630" },
            { property: "og:image:alt", content: "Cardstack: track every card you own" },
            { name: "twitter:image", content: `${SITE_URL}/og-image.png` },
          ]
        : []),
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
        <ThemeProvider>
          {children}
          <Toaster />
        </ThemeProvider>
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
