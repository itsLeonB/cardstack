import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"

/**
 * The ordinary not-found page: an unknown address, and any address for a Guest
 * or a signed-in user who is not listed, so the Admin app does not show that it
 * exists. No links, since every path here is the Admin app's own.
 */
export function NotFound() {
  return (
    <PageContainer variant="narrow">
      <PageHeader
        title="Page not found"
        description="The page you're looking for doesn't exist or has moved."
      />
    </PageContainer>
  )
}
