import { Link, createFileRoute } from "@tanstack/react-router"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { clerkAuth } from "@/lib/clerk-auth"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/_admin/")({
  head: () => pageHead("Catalog admin"),
  component: AdminHome,
})

function AdminHome() {
  return (
    <PageContainer>
      <PageHeader
        title="Catalog admin"
        actions={
          <Button
            variant="outline"
            onClick={() =>
              clerkAuth.endSession(() => window.location.assign("/"))
            }
          >
            Sign out
          </Button>
        }
      />
      <Link to="/expansion-sets" className="text-foreground underline">
        Expansion Sets
      </Link>
      <Link to="/cards" className="text-foreground underline">
        Cards
      </Link>
    </PageContainer>
  )
}
