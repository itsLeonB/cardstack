import { Link } from "@tanstack/react-router"
import { buttonVariants } from "@/components/ui/button"
import { scanEnabled } from "@/lib/scan-flag"

/** "Scan cards" on a Collection's page; absent while the feature flag is off. */
export function ScanLink({ collectionId }: { collectionId: string }) {
  if (!scanEnabled()) return null
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
