import { createFileRoute } from "@tanstack/react-router"
import { ReviewQueue } from "@/components/admin/review-queue"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/_admin/reviews")({
  head: () => pageHead("Corrections to review"),
  component: ReviewsPage,
})

function ReviewsPage() {
  return (
    <PageContainer>
      <PageHeader
        title="Corrections to review"
        description="Approve a photo that shows the chosen Card, or reject one that does not. Photos stay in the bucket."
      />
      <ReviewQueue />
    </PageContainer>
  )
}
