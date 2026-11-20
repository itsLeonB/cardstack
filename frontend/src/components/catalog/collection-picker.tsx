import { useEffect } from "react"
import { SignInLink } from "@/components/auth/sign-in-link"
import { useListCollections } from "@/generated/endpoints/collections/collections"
import { useSession } from "@/lib/session"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

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

  // Only definitive answers clear the selection: not loading, 5xx or network errors.
  const listStatus = query.data?.status
  const denied = (status?: number) => status === 401 || status === 403
  const invalid =
    selectedId !== undefined &&
    ((!isLoading && !isAuthenticated) ||
      denied(listStatus) ||
      listStatus === 404 ||
      (loaded && !known))
  useEffect(() => {
    if (invalid) onSelect(undefined)
  }, [invalid, onSelect])

  return {
    isAuthenticated,
    isLoading,
    collections,
    selected: known && isAuthenticated ? selectedId : undefined,
  }
}

/** "Add to collection" selector; disabled with a login prompt for guests. */
export function CollectionPicker({
  isAuthenticated,
  isLoading = false,
  loginRedirect = "/catalog/search",
  collections,
  value,
  onChange,
}: {
  isAuthenticated: boolean
  isLoading?: boolean
  /** Same-origin path (with search) to return to after logging in. */
  loginRedirect?: string
  collections: { id: string; title: string }[]
  value: string | undefined
  onChange: (id: string | undefined) => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <label htmlFor="add-to-collection" className="text-sm font-medium">
        Add to collection
      </label>
      <Select
        value={value ?? ""}
        onValueChange={(next) => onChange(next || undefined)}
        disabled={!isAuthenticated}
      >
        <SelectTrigger id="add-to-collection">
          <SelectValue placeholder="None" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="">None</SelectItem>
          {collections.map((collection) => (
            <SelectItem key={collection.id} value={collection.id}>
              {collection.title}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {!isAuthenticated && !isLoading && (
        <span className="text-sm">
          <SignInLink redirect={loginRedirect}>
            Sign in to add cards to a Collection
          </SignInLink>
        </span>
      )}
    </div>
  )
}
