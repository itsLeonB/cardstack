import { SiteFooter } from "./site-footer"
import { SiteHeader } from "./site-header"
import { SkipLink } from "./skip-link"

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-svh flex-col">
      <SkipLink />
      <SiteHeader />
      <main id="main-content" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </main>
      <SiteFooter />
    </div>
  )
}
