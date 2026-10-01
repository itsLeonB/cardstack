import { useNavLinks } from "./nav-links"
import { NavLinkItem } from "./site-header"
import { Wordmark } from "./wordmark"

export function SiteFooter() {
  const links = useNavLinks()

  return (
    <footer className="mt-12 border-t">
      <div className="mx-auto flex max-w-5xl flex-col gap-4 px-4 py-8 text-sm text-muted-foreground">
        <Wordmark />
        <p>Track every card you own, across every binder.</p>
        <nav aria-label="Footer" className="-ml-3 flex flex-wrap gap-1">
          {links.map((link) => (
            <NavLinkItem key={link.to} link={link} />
          ))}
        </nav>
        <p>
          A personal MVP covering Indonesian print editions of Pokémon cards.
        </p>
      </div>
    </footer>
  )
}
