import { Link } from "@tanstack/react-router"
import { buttonVariants } from "@/components/ui/button"

export function AddCardsLink({ collectionId }: { collectionId: string }) {
  return (
    <Link to="/catalog/search" search={{ collectionId }} className={buttonVariants()}>
      Add Cards
    </Link>
  )
}
