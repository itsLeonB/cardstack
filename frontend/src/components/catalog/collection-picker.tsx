import { useEffect } from "react"
import { Link } from "@tanstack/react-router"
import { useListCollections } from "@/generated/endpoints/collections/collections"
import { useSession } from "@/lib/session"

/**
 * Session + the user's Collections for the public catalog (no redirecting
 * guard). `selected` is the URL's Collection id only while it is valid; once
 * the session or the Collection list says otherwise, `onSelect(undefined)`
 * clears it.
 */
export function useCatalogCollection(
  selectedId: string | undefined,
  onSelect: (id: string | undefined) => void
) {
  const { isAuthenticated, isLoading } = useSession()
  const query = useListCollections({ query: { enabled: isAuthenticated } })
  const collections =
    query.data?.status === 200 ? (query.data.data.data ?? []) : []
  const loaded = query.data?.status === 200
  const known = collections.some((collection) => collection.id === selectedId)

  const invalid =
    selectedId !== undefined &&
    ((!isLoading && !isAuthenticated) || (loaded && !known))
  useEffect(() => {
    if (invalid) onSelect(undefined)
  }, [invalid, onSelect])

  return {
    isAuthenticated,
    collections,
    selected: known && isAuthenticated ? selectedId : undefined,
  }
}

/** "Add to collection" selector; disabled with a login prompt for guests. */
export function CollectionPicker({
  isAuthenticated,
  collections,
  value,
  onChange,
}: {
  isAuthenticated: boolean
  collections: { id: string; title: string }[]
  value: string | undefined
  onChange: (id: string | undefined) => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <label htmlFor="add-to-collection" className="text-sm font-medium">
        Add to collection
      </label>
      <select
        id="add-to-collection"
        disabled={!isAuthenticated}
        value={value ?? ""}
        onChange={(event) => onChange(event.target.value || undefined)}
        className="h-9 rounded-3xl border border-transparent bg-input/50 px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <option value="">None</option>
        {collections.map((collection) => (
          <option key={collection.id} value={collection.id}>
            {collection.title}
          </option>
        ))}
      </select>
      {!isAuthenticated && (
        <Link
          to="/login"
          search={{ redirect: "/catalog/search" }}
          className="text-sm underline-offset-2 hover:underline"
        >
          Log in to add cards to a Collection
        </Link>
      )}
    </div>
  )
}
