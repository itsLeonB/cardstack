import { useState } from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { useDeleteCollectionMutation } from "@/lib/collections"

interface DeleteCollectionDialogProps {
  collectionId: string
  collectionTitle: string
}

export function DeleteCollectionDialog({
  collectionId,
  collectionTitle,
}: DeleteCollectionDialogProps) {
  const [open, setOpen] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const deleteMutation = useDeleteCollectionMutation()

  function handleConfirm() {
    setErrorMessage(null)
    deleteMutation.mutate(
      { id: collectionId },
      {
        onSuccess: (response) => {
          if (response.status === 204) {
            setOpen(false)
            return
          }
          setErrorMessage(response.data.detail ?? "Could not delete this Collection.")
        },
        onError: () => {
          setErrorMessage("Could not reach the server. Please try again.")
        },
      }
    )
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (nextOpen) setErrorMessage(null)
      }}
    >
      <AlertDialogTrigger
        render={<Button variant="destructive" size="sm" />}
        aria-label={`Delete ${collectionTitle}`}
      >
        Delete
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete &ldquo;{collectionTitle}&rdquo;?</AlertDialogTitle>
          <AlertDialogDescription>
            This permanently deletes the Collection. This cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {errorMessage && (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage}
          </p>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleteMutation.isPending}
            onClick={handleConfirm}
          >
            {deleteMutation.isPending ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
