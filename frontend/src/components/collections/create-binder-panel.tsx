import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useCreateBinder } from "@/generated/endpoints/binder/binder"
import { BinderLayoutForm } from "@/components/collections/binder-layout-form"
import type { BinderLayoutValues } from "@/components/collections/binder-layout-form"
import { invalidateBinder } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

/** Turns a plain, empty Collection into a binder. */
export function CreateBinderPanel({ collectionId }: { collectionId: string }) {
  const queryClient = useQueryClient()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const createMutation = useCreateBinder()

  function createBinder(values: BinderLayoutValues) {
    setErrorMessage(null)
    createMutation.mutate(
      {
        id: collectionId,
        data: {
          rows: values.rows,
          columns: values.columns,
          leafCount: values.leafCount,
        },
      },
      {
        onSuccess: (response) => {
          if (response.status !== 201) {
            setErrorMessage(
              errorDetail(response, "Could not create this binder.") ?? null
            )
            return
          }
          invalidateBinder(queryClient, collectionId)
        },
        onError: () => setErrorMessage(NETWORK_ERROR),
      }
    )
  }

  return (
    <section aria-label="Make a binder" className="flex flex-col gap-4">
      <h2 className="text-lg font-medium">Make this a binder</h2>
      <p className="text-sm">
        Choose the Layout of each Page and how many Leaves the binder starts
        with. Its Slots then hold the cards.
      </p>
      <div className="max-w-lg">
        <BinderLayoutForm
          withLeafCount
          submitLabel="Create binder"
          pendingLabel="Creating..."
          isPending={createMutation.isPending}
          errorMessage={errorMessage}
          onSubmit={createBinder}
        />
      </div>
    </section>
  )
}
