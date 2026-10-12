import { ExpansionSetForm } from "@/components/admin/expansion-set-form"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"

/** The body of `/admin`. Later tickets add the card form here. */
export function AdminPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Catalog admin"
        description="Enter and correct Expansion Sets and Cards by hand."
      />
      <section aria-labelledby="admin-sets-heading" className="max-w-xl">
        <h2 id="admin-sets-heading" className="mb-4 text-lg font-medium">
          Expansion Set
        </h2>
        <ExpansionSetForm />
      </section>
    </PageContainer>
  )
}
