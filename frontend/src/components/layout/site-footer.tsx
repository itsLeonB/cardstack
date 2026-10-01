import { Wordmark } from "./wordmark"

export function SiteFooter() {
  return (
    <footer className="mt-12 border-t">
      <div className="mx-auto flex max-w-5xl flex-col gap-4 px-4 py-8 text-sm text-muted-foreground">
        <Wordmark />
        <p>Track every card you own, across every binder.</p>
        <p>
          A personal MVP covering Indonesian print editions of Pokémon cards.
        </p>
      </div>
    </footer>
  )
}
