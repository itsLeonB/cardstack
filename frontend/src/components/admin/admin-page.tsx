import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"

/** The body of `/admin`. Later tickets add the set and card forms here. */
export function AdminPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Catalog admin"
        description="Enter and correct Expansion Sets and Cards by hand."
      />
    </PageContainer>
  )
}
