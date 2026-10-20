import { createFileRoute } from "@tanstack/react-router"
import { ExpansionSetForm } from "@/components/admin/expansion-set-form"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/_admin/expansion-sets")({
  head: () => pageHead("Expansion Sets"),
  component: ExpansionSetsPage,
})

function ExpansionSetsPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Expansion Sets"
        description="Create an Expansion Set, or pick one to edit it."
      />
      <section aria-labelledby="expansion-set-heading" className="max-w-xl">
        <h2 id="expansion-set-heading" className="mb-4 text-lg font-medium">
          Expansion Set
        </h2>
        <ExpansionSetForm />
      </section>
    </PageContainer>
  )
}
