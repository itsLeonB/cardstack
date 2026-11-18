import { Link } from "@tanstack/react-router"
import { RiCameraLine } from "@remixicon/react"
import { buttonVariants } from "@/components/ui/button"

/** "Scan cards" on a Collection's page. */
export function ScanLink({ collectionId }: { collectionId: string }) {
  return (
    <Link
      to="/collections/$collectionId/scan"
      params={{ collectionId }}
      className={`${buttonVariants({ variant: "outline" })} min-h-11 whitespace-normal`}
    >
      <RiCameraLine aria-hidden="true" data-icon="inline-start" />
      Scan cards
    </Link>
  )
}
