# 02: Share a Collection and open its link (list view)

**What to build:** an owner makes a Collection Shared with a share-icon button in its header, copies the link, and anyone, a Guest included, opens `/s/<profile slug>/<collection slug>` and sees the Collection read-only as a list of cards. Sharing is off by default and turning it off ends the link at once (GLOSSARY.md: Shared, Slug; ADR-0025).

**Blocked by:** 01 (Profile slugs).

**Status:** done

- [x] A Collection has a nullable Slug, unique within its owner. Turning sharing on makes it from the current title plus the last 5 hex characters of a fresh random uuid (not the Collection's id, so a revoked link never comes back), with the fallback stem `collection`. It stays fixed while Shared, so a rename does not break the link. Turning sharing off clears it to NULL, and sharing again makes a fresh one from the current title.
- [x] A guest-allowed public route group with its own rate limit resolves a profile Slug and then a Collection Slug. It serves the owner's display name, the title and description, and the entries with quantities, with the same filter parameters and facets as the owner's listing, including the rarity, category and tag filters and facets for Guests. It never returns the owner's email.
- [x] A private, unshared or nonexistent target gives one identical not-found response. Cards in unpublished Expansion Sets are left out. Deleting a Shared Collection ends its link.
- [x] The Share button, with a share icon and a label that shows whether the Collection is Shared, opens a dialog with the name warning above the switch before first use, the "Anyone with the link can view" switch, the read-only link with a Copy button and an announced "Copied" confirmation, and a confirmation when turning sharing off ("Existing links will stop working"). The dialog is one reusable component, since the Wishlist uses it later.
- [x] The shared page lives outside the signed-in layout, shows "Shared by <name>", the title and description, and the read-only card list with filters. It has no edit controls and carries `noindex`.
- [x] HTTP-boundary tests cover share, unshare, re-share, rename while Shared, the Guest, signed-in and owner views, the identical not-found, the missing email, unpublished sets and deleting a Shared Collection. Screen tests with the network mocked cover the dialog and the shared page. One guest Playwright spec opens a Shared Collection link.
