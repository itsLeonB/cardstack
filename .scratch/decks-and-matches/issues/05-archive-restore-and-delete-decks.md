# 05: Archive, restore and delete Decks

**What to build:** Let an owner stop using a Deck without losing match history, bring it back later, or permanently remove an unused Deck.

**Blocked by:** 02: Record, resume and finish matches.

**Status:** ready-for-agent

- [ ] Archive a personal Deck and persist its archived state. Exclude it from choices for a new match without removing its existing In Progress or Finished matches or their saved Deck details.
- [ ] Restore an archived Deck and make it available for new matches again, without changing its identity or existing match references.
- [ ] Permanently delete only a Deck with no match records, whether the Deck is active or archived. Reject deletion of a Deck referenced by any In Progress or Finished match; archiving is the available alternative.
- [ ] Existing matches remain viewable and resumable after archiving. Archiving must not prevent their correction when ticket 03 becomes available, and no archive, restore or delete operation changes Collections or owned quantities. Only the owner may perform these operations.
- [ ] Automated checks cover archive/restore persistence, exclusion from new-match choices, used-Deck deletion rejection, unused-Deck deletion and owner isolation. A browser journey archives a Deck, resumes an existing match and restores the Deck. This ticket does not depend on history filters or completed-match corrections.
