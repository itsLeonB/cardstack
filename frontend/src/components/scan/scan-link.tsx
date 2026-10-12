import { Link } from "@tanstack/react-router"
import { buttonVariants } from "@/components/ui/button"

/** "Scan cards" on a Collection's page. */
export function ScanLink({ collectionId }: { collectionId: string }) {
  return (
    <Link
      to="/collections/$collectionId/scan"
      params={{ collectionId }}
      className={buttonVariants({ variant: "outline" })}
    >
      Scan cards
    </Link>
  )
}
