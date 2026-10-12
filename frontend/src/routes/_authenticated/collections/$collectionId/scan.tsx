import { Link, createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { notFoundResource } from "@/components/layout/not-found"
import { PageHeader } from "@/components/layout/page-header"
import { buttonVariants } from "@/components/ui/button"
import { ScanScreen } from "@/components/scan/scan-screen"
import { getGetCollectionQueryOptions } from "@/generated/endpoints/collections/collections"
import { clearDraft } from "@/lib/draft-addition"
import { useCameraFrameSource } from "@/lib/frame-source"

export const Route = createFileRoute(
  "/_authenticated/collections/$collectionId/scan"
)({
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
    <>
      {/* The viewfinder takes the page's place, so the title is for screen readers only. */}
      <div className="sr-only">
        <PageHeader
          title="Scan cards"
          description={title && `Adding to ${title}`}
        />
      </div>
      <ScanScreen
        key={collectionId}
        collectionId={collectionId}
        source={source}
        backLink={
          <Link
            to="/collections/$collectionId"
            params={{ collectionId }}
            className={buttonVariants({ variant: "secondary", size: "sm" })}
          >
            {title ? `Back to ${title}` : "Back to Collection"}
          </Link>
        }
        onAdded={() =>
          void navigate({
            to: "/collections/$collectionId",
            params: { collectionId },
          })
        }
      />
    </>
  )
}
