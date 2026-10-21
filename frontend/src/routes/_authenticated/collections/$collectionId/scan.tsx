import { Link, createFileRoute, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { notFoundResource } from "@/components/layout/not-found"
import { PageHeader } from "@/components/layout/page-header"
import { buttonVariants } from "@/components/ui/button"
import { ScanScreen } from "@/components/scan/scan-screen"
import { getGetCollectionQueryOptions } from "@/generated/endpoints/collections/collections"
import { useCameraFrameSource } from "@/lib/frame-source"
import { useSession } from "@/lib/session"

export const Route = createFileRoute(
  "/_authenticated/collections/$collectionId/scan"
)({
  loader: async ({ context: { queryClient }, params }) => {
    const response = await queryClient.ensureQueryData(
      getGetCollectionQueryOptions(params.collectionId)
    )
    if (response.status === 404) {
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
  const { user } = useSession()
  // The route is signed in only, so the draft always has an owner.
  if (!user) return null

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
        key={`${user.id}:${collectionId}`}
        userId={user.id}
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
