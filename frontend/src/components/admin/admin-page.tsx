import { CardForm } from "@/components/admin/card-form"
import { DeleteCardForm } from "@/components/admin/delete-card-form"
import { ExpansionSetForm } from "@/components/admin/expansion-set-form"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"

/** The body of `/admin`: the Expansion Set form, then the Card form. */
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
      <section aria-labelledby="admin-cards-heading" className="mt-10 max-w-xl">
        <h2 id="admin-cards-heading" className="mb-4 text-lg font-medium">
          Card
        </h2>
        <CardForm />
      </section>
      <section
        aria-labelledby="admin-delete-heading"
        className="mt-10 max-w-xl"
      >
        <h2 id="admin-delete-heading" className="mb-4 text-lg font-medium">
          Delete a card
        </h2>
        <DeleteCardForm />
      </section>
    </PageContainer>
  )
}
