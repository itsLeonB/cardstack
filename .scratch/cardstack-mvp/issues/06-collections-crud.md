# 06: Collections CRUD

**What to build:** An authenticated user can create, list, edit, and delete their own Collections (binders/boxes/decks), each with a title, optional description, and optional hard capacity limit. Deletes require explicit confirmation since they're irreversible (no soft delete).

**Blocked by:** 02 (Auth: registration & login)

**Status:** ready-for-agent

- [ ] Authenticated user can create a Collection with a title, optional description, and optional maximum card-count limit (summed quantity, not distinct-card count)
- [ ] Authenticated user can list their own Collections
- [ ] Authenticated user can edit a Collection's title/description/limit
- [ ] Authenticated user can delete a Collection
- [ ] Frontend requires explicit confirmation before a delete request is sent (hard delete, no undo)
- [ ] A user cannot view, edit, or delete another user's Collection
