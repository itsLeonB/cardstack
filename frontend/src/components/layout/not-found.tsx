import { Link, notFound } from "@tanstack/react-router"
import type { NotFoundRouteProps } from "@tanstack/react-router"
import { z } from "zod"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"

const notFoundData = z.object({ resource: z.enum(["Collection", "Card"]) })
type Resource = z.infer<typeof notFoundData>["resource"]

/** Throw from a loader when the entity behind the URL does not exist; `NotFound` names it. */
export function notFoundResource(resource: Resource) {
  return notFound({ data: { resource } })
}

/** Not-found view for unknown URLs and missing entities; rendered inside whichever shell it bubbles to. */
export function NotFound({ data }: NotFoundRouteProps) {
  const resource = notFoundData.safeParse(data).data?.resource
  return (
    <PageContainer variant="narrow">
      <PageHeader
        title={resource ? `${resource} not found` : "Page not found"}
        description={
          resource
            ? `We couldn't find that ${resource}. It may have been deleted, or the link may be wrong.`
            : "The page you're looking for doesn't exist or has moved."
        }
      />
      <div className="flex flex-col gap-3 sm:flex-row">
        <Button render={<Link to="/" />}>Home</Button>
        <Button variant="outline" render={<Link to="/catalog" />}>
          Catalog
        </Button>
      </div>
    </PageContainer>
  )
}
