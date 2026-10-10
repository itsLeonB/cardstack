import { createFileRoute, notFound, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { Breadcrumbs } from "@/components/layout/breadcrumbs"
import { notFoundResource } from "@/components/layout/not-found"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { ScanScreen } from "@/components/scan/scan-screen"
import { getGetCollectionQueryOptions } from "@/generated/endpoints/collections/collections"
import { clearDraft } from "@/lib/draft-addition"
import { useCameraFrameSource } from "@/lib/frame-source"
import { scanEnabled } from "@/lib/scan-flag"

export const Route = createFileRoute(
  "/_authenticated/collections/$collectionId/scan"
)({
  // With the flag off the route does not exist.
  beforeLoad: () => {
    if (!scanEnabled()) throw notFound()
  },
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    )
    if (response.status === 404) {
      // An orphan draft is dropped quietly.
      clearDraft(params.collectionId)
      throw notFoundResource("Collection")
    }
    return response.status === 200 ? response.data.data.title : undefined
  },
  head: () => pageHead("Scan cards"),
  component: ScanPage,
})

function ScanPage() {
  const { collectionId } = Route.useParams()
  const title = Route.useLoaderData()
  const source = useCameraFrameSource()
  const navigate = useNavigate()

  return (
    <PageContainer variant="narrow">
      <Breadcrumbs
        crumbs={[
          { label: "Collections", link: { to: "/collections" } },
          {
            label: title ?? "Collection",
            link: {
              to: "/collections/$collectionId",
              params: { collectionId },
            },
          },
          { label: "Scan cards" },
        ]}
      />
      <PageHeader
        title="Scan cards"
        description={title && `Adding to ${title}`}
      />
      <ScanScreen
        key={collectionId}
        collectionId={collectionId}
        source={source}
        onAdded={() =>
          void navigate({
            to: "/collections/$collectionId",
            params: { collectionId },
          })
        }
      />
    </PageContainer>
  )
}
