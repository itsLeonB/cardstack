import { SkipLink } from "@/components/layout/skip-link"
import { ThemeToggle } from "@/components/layout/theme-toggle"
import { Wordmark } from "@/components/layout/wordmark"

/** Minimal shell for everything under `routes/auth/`: wordmark and theme toggle in a bare `header` (no nav) and no footer. Provides the page's single `main`. */
export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-svh flex-col">
      <SkipLink />
      <header className="mx-auto flex w-full max-w-sm items-center justify-between py-4 pr-[max(1rem,env(safe-area-inset-right,0px))] pl-[max(1rem,env(safe-area-inset-left,0px))] sm:pr-[max(1.5rem,env(safe-area-inset-right,0px))] sm:pl-[max(1.5rem,env(safe-area-inset-left,0px))]">
        <Wordmark />
        <ThemeToggle />
      </header>
      <main
        id="main-content"
        tabIndex={-1}
        className="flex flex-1 flex-col justify-center pb-16 outline-none"
      >
        {children}
      </main>
    </div>
  )
}
