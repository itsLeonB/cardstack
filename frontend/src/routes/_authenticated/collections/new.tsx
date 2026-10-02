import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Card, CardContent } from "@/components/ui/card"
import { CollectionForm } from "@/components/collections/collection-form"
import {
  collectionSubmitCallbacks,
  useCreateCollectionMutation,
} from "@/lib/collections"

export const Route = createFileRoute("/_authenticated/collections/new")({
  head: () => pageHead("New Collection"),
  component: NewCollectionPage,
})

function NewCollectionPage() {
  const navigate = useNavigate()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const createMutation = useCreateCollectionMutation()

  return (
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          { label: "Collections", link: { to: "/collections" } },
          { label: "New Collection" },
        ]}
      />
      <PageHeader
        title="New Collection"
        description="Give it a title, and optionally a description or a hard cap on its summed card quantity."
      />

      <Card className="max-w-lg">
        <CardContent>
          <CollectionForm
            submitLabel="Create Collection"
            pendingLabel="Creating..."
            isPending={createMutation.isPending}
            errorMessage={errorMessage}
            onSubmit={(body) => {
              setErrorMessage(null)
              createMutation.mutate(
                { data: body },
                collectionSubmitCallbacks({
                  successStatus: 201,
                  failureMessage: "Could not create this Collection.",
                  onDone: () => void navigate({ to: "/collections" }),
                  setErrorMessage,
                })
              )
            }}
          />
        </CardContent>
      </Card>
    </PageContainer>
  )
}
