import { createFileRoute } from "@tanstack/react-router"
import { CardForm } from "@/components/admin/card-form"
import { DeleteCardForm } from "@/components/admin/delete-card-form"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/_admin/cards")({
  head: () => pageHead("Cards"),
  component: CardsPage,
})

function CardsPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Cards"
        description="Add or correct a Card and its image, or delete one."
      />
      <section aria-labelledby="card-heading" className="max-w-xl">
        <h2 id="card-heading" className="mb-4 text-lg font-medium">
          Card
        </h2>
        <CardForm />
      </section>
      <section aria-labelledby="delete-card-heading" className="mt-10 max-w-xl">
        <h2 id="delete-card-heading" className="mb-4 text-lg font-medium">
          Delete a card
        </h2>
        <DeleteCardForm />
      </section>
    </PageContainer>
  )
}
