import { Link } from "@tanstack/react-router"
import { RiCameraLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"

/** "Scan cards" on a Collection's page. */
export function ScanLink({ collectionId }: { collectionId: string }) {
  return (
    <Button
      variant="outline"
      className="h-auto min-h-11 whitespace-normal"
      render={
        <Link to="/collections/$collectionId/scan" params={{ collectionId }} />
      }
    >
      <RiCameraLine aria-hidden="true" data-icon="inline-start" />
      Scan cards
    </Button>
  )
}
