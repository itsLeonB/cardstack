import { Link } from "@tanstack/react-router"
import { RiAddLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"

export function AddCardsLink({ collectionId }: { collectionId: string }) {
  return (
    <Button
      variant="outline"
      className="h-auto min-h-11 whitespace-normal"
      render={<Link to="/catalog/search" search={{ collectionId }} />}
    >
      <RiAddLine aria-hidden="true" data-icon="inline-start" />
      Add Cards
    </Button>
  )
}
