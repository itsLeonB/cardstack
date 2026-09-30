import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { CollectionForm } from "@/components/collections/collection-form"
import {
  collectionSubmitCallbacks,
  useCreateCollectionMutation,
} from "@/lib/collections"

export const Route = createFileRoute("/_authenticated/collections/new")({
  component: NewCollectionPage,
})

function NewCollectionPage() {
  const navigate = useNavigate()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const createMutation = useCreateCollectionMutation()

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
          <CardTitle>New Collection</CardTitle>
          <CardDescription>
            Give it a title, and optionally a description or a hard cap on
            its summed card quantity.
          </CardDescription>
        </CardHeader>
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
    </main>
  )
}
