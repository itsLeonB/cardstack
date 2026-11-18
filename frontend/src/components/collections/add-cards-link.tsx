import { Link } from "@tanstack/react-router"
import { RiAddLine } from "@remixicon/react"
import { buttonVariants } from "@/components/ui/button"

export function AddCardsLink({ collectionId }: { collectionId: string }) {
  return (
    <Link
      to="/catalog/search"
      search={{ collectionId }}
      className={`${buttonVariants()} min-h-11 whitespace-normal`}
    >
      <RiAddLine aria-hidden="true" data-icon="inline-start" />
      Add Cards
    </Link>
  )
}
