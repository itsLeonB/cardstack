import { useState } from "react"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import {
  getGetCollectionQueryOptions,
  useGetCollection,
} from "@/generated/endpoints/collections/collections"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { CollectionForm } from "@/components/collections/collection-form"
import {
  collectionSubmitCallbacks,
  errorDetail,
  useUpdateCollectionMutation,
} from "@/lib/collections"

export const Route = createFileRoute(
  "/_authenticated/collections/$collectionId/edit"
)({
  loader: ({ context: { queryClient }, params }) =>
    queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    ),
  component: EditCollectionPage,
})

function EditCollectionPage() {
  const { collectionId } = Route.useParams()
  const navigate = useNavigate()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const query = useGetCollection(collectionId)
  const collection = query.data?.status === 200 ? query.data.data.data : undefined
  const collectionTitle = collection?.title ?? "Collection"
  const loadErrorMessage = errorDetail(
    query.data,
    "Could not load this Collection."
  )

  const updateMutation = useUpdateCollectionMutation()

  return (
    <PageContainer>
      <Breadcrumbs
        crumbs={[
          { label: "Collections", link: { to: "/collections" } },
          {
            label: collectionTitle,
            link: { to: "/collections/$collectionId", params: { collectionId } },
          },
          { label: "Edit" },
        ]}
      />
      <PageHeader
        title={collection ? `Edit “${collectionTitle}”` : "Edit Collection"}
        description="Update the title, description, or card-count limit."
      />

      <Card className="max-w-lg">
        <CardContent>
          {query.isPending && (
            <div
              className="flex flex-col gap-4"
              aria-busy="true"
              aria-label="Loading Collection"
            >
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          )}

          {(query.isError || loadErrorMessage) && (
            <p role="alert" className="text-sm text-destructive">
              {loadErrorMessage ?? "Could not reach the backend. Please try again."}
            </p>
          )}

          {collection && (
            <CollectionForm
              initialValues={{
                title: collection.title,
                description: collection.description,
                maxCardCount: collection.maxCardCount > 0 ? collection.maxCardCount.toString() : "",
              }}
              submitLabel="Save changes"
              pendingLabel="Saving..."
              isPending={updateMutation.isPending}
              errorMessage={errorMessage}
              onSubmit={(body) => {
                setErrorMessage(null)
                updateMutation.mutate(
                  { id: collectionId, data: body },
                  collectionSubmitCallbacks({
                    successStatus: 200,
                    failureMessage: "Could not save this Collection.",
                    onDone: () => void navigate({ to: "/collections" }),
                    setErrorMessage,
                  })
                )
              }}
            />
          )}
        </CardContent>
      </Card>
    </PageContainer>
  )
}
