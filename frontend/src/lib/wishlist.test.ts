import { QueryClient } from "@tanstack/react-query"
import { expect, it } from "vitest"
import { invalidateWishlist } from "./wishlist"
import {
  getListWishlistFacetsQueryKey,
  getListWishlistInfiniteQueryKey,
  getListWishlistQueryKey,
} from "@/generated/endpoints/wishlist/wishlist"

// Real keys in a real cache: adding a Card from the catalog must refresh every
// shape of the Wishlist's data, including the lookup a catalog result reads.
it("invalidates the infinite list, the plain list and the facets", () => {
  const client = new QueryClient()
  const keys = [
    getListWishlistInfiniteQueryKey({ status: "all", limit: 60 }),
    getListWishlistQueryKey({ cardId: ["card-1"], limit: 1 }),
    getListWishlistFacetsQueryKey({ status: "all" }),
  ]
  for (const key of keys) client.setQueryData(key, {})

  invalidateWishlist(client)

  for (const key of keys)
    expect(client.getQueryState(key)?.isInvalidated).toBe(true)
})
