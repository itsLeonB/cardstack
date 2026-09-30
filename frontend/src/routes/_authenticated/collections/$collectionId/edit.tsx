import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import {
  getGetCollectionQueryOptions,
  useGetCollection,
} from "@/generated/endpoints/collections/collections"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { CollectionForm } from "@/components/collections/collection-form"
import { useUpdateCollectionMutation } from "@/lib/collections"

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
  const loadErrorMessage =
    query.data && query.data.status !== 200
      ? (query.data.data.detail ?? "Could not load this Collection.")
      : undefined

  const updateMutation = useUpdateCollectionMutation()

  return (
    <main className="mx-auto flex max-w-lg flex-col gap-6 p-6">
      <Link
        to="/collections"
        className="w-fit text-sm text-muted-foreground underline-offset-2 hover:underline"
      >
        ← Back to Collections
      </Link>

      <Card>
        <CardHeader>
          <CardTitle>
            {collection ? `Edit “${collection.title}”` : "Edit Collection"}
          </CardTitle>
          <CardDescription>
            Update the title, description, or card-count limit.
          </CardDescription>
        </CardHeader>
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
                  {
                    onSuccess: (response) => {
                      if (response.status === 200) {
                        void navigate({ to: "/collections" })
                        return
                      }
                      setErrorMessage(
                        response.data.detail ?? "Could not save this Collection."
                      )
                    },
                    onError: () => {
                      setErrorMessage(
                        "Could not reach the server. Please try again."
                      )
                    },
                  }
                )
              }}
            />
          )}
        </CardContent>
      </Card>
    </main>
  )
}
