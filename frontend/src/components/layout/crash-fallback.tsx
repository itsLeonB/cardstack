import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"

/**
 * Router `defaultErrorComponent`. Deliberately ignores the error it is given:
 * stack traces and raw messages never reach the screen.
 */
export function CrashFallback() {
  return (
    <PageContainer variant="narrow">
      <PageHeader
        title="Something went wrong"
        description="An unexpected error stopped this page from loading. Reloading usually fixes it."
        actions={<Button onClick={() => window.location.reload()}>Reload</Button>}
      />
    </PageContainer>
  )
}
