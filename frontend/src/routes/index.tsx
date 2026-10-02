import { createFileRoute } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { HomePage } from "@/components/home/home-page"

export const Route = createFileRoute("/")({
  head: () => pageHead("Track every Pokémon card you own"),
  component: HomePage,
})
