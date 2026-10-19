import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
} from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"

import { AppClerkProvider } from "@/components/auth/clerk-provider"
import { NotFound } from "@/components/layout/not-found"
import { ThemeProvider } from "@/components/theme-provider"
import type { AuthGate } from "@/lib/clerk-auth"
import { NOINDEX_META, SITE_NAME } from "@/lib/site"
import appCss from "../styles.css?url"

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
  auth: AuthGate
}>()({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { title: SITE_NAME },
      // Also sent as X-Robots-Tag by vercel.json; the meta tag covers hosts
      // that ignore headers.
      NOINDEX_META,
    ],
    links: [
      { rel: "icon", href: "/favicon.svg", type: "image/svg+xml" },
      { rel: "stylesheet", href: appCss },
    ],
  }),
  notFoundComponent: NotFound,
  component: RootLayout,
  shellComponent: RootDocument,
})

function RootLayout() {
  return (
    <div className="flex min-h-svh flex-col">
      <main className="flex-1">
        <Outlet />
      </main>
    </div>
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
          <AppClerkProvider>{children}</AppClerkProvider>
        </ThemeProvider>
        <Scripts />
      </body>
    </html>
  )
}
