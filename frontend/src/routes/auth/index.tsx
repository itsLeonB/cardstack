import { createFileRoute, notFound } from "@tanstack/react-router"
import { NotFound } from "@/components/layout/not-found"

// `/auth` itself isn't a page; show the not-found view inside the auth shell, not an empty one.
export const Route = createFileRoute("/auth/")({
  beforeLoad: () => {
    throw notFound()
  },
  notFoundComponent: NotFound,
})
