import { SkipLink } from "@/components/layout/skip-link"
import { ThemeToggle } from "@/components/layout/theme-toggle"
import { Wordmark } from "@/components/layout/wordmark"

/** Minimal shell for everything under `routes/auth/`: wordmark and theme toggle in a bare `header` (no nav) and no footer. Provides the page's single `main`. */
export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-svh flex-col">
      <SkipLink />
      <header className="mx-auto flex w-full max-w-sm items-center justify-between px-4 py-4 sm:px-6">
        <Wordmark />
        <ThemeToggle />
      </header>
      <main id="main-content" tabIndex={-1} className="flex flex-1 flex-col justify-center pb-16 outline-none">
        {children}
      </main>
    </div>
  )
}
