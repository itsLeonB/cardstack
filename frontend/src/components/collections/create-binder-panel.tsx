import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { useCreateBinder } from "@/generated/endpoints/binder/binder"
import { BinderLayoutForm } from "@/components/collections/binder-layout-form"
import type { BinderLayoutValues } from "@/components/collections/binder-layout-form"
import { invalidateBinder } from "@/lib/binder"
import { errorDetail, NETWORK_ERROR } from "@/lib/collections"

/** Turns a plain, empty Collection into a binder. */
export function CreateBinderPanel({ collectionId }: { collectionId: string }) {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const createMutation = useCreateBinder({
    mutation: {
      onSuccess: (response, variables) => {
        if (response.status === 201) {
          invalidateBinder(queryClient, variables.id)
        }
      },
    },
  })

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
          setOpen(false)
        },
        onError: () => setErrorMessage(NETWORK_ERROR),
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" className="min-h-11" />}>
        Make this a binder
      </DialogTrigger>
      <DialogContent
        showCloseButton={false}
        className="max-h-[calc(100dvh-2rem)] overflow-y-auto"
      >
        <DialogHeader>
          <DialogTitle>Make this a binder</DialogTitle>
          <DialogDescription>
            Choose the Layout of each Page and how many Leaves the binder starts
            with. Its Slots then hold the cards.
          </DialogDescription>
        </DialogHeader>
        <div className="max-w-lg">
          <BinderLayoutForm
            withLeafCount
            submitLabel="Create binder"
            pendingLabel="Creating..."
            isPending={createMutation.isPending}
            errorMessage={errorMessage}
            onCancel={() => setOpen(false)}
            onSubmit={createBinder}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
