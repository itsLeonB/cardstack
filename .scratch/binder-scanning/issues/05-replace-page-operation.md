# 05: Replace Page operation

**What to build:** One authenticated backend operation replaces the whole contents of a Page. The request names the Page and carries one state per Slot in grid order: empty, a Card id, or Unidentified. It is idempotent on retry. A Card id the server does not know is stored as Unidentified, and the response reports that for that Slot. A Page that does not exist, or a request whose Slot count does not match the Layout, is rejected as a whole and changes nothing. The response gives a result per Slot (stored as sent, or stored Unidentified with the reason). The Inventory Entries of every Card the Page held before or holds after are updated in the same transaction so that they equal the Slot counts.

Spec: `.scratch/binder-scanning/spec.md` (Replace Page, Quantity invariant). The page-scan screen (ticket 06) is the first caller.

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** ready-for-agent

- [ ] Replacing a Page sets each Slot to empty, the given Card, or Unidentified, and cards that were on the Page and are no longer there are removed from it
- [ ] Sending the same request twice gives the same state and the same response
- [ ] A Card id the server does not know is stored as Unidentified and reported for that Slot, and the other Slots are still applied
- [ ] A Page number that does not exist, or a Slot count that does not match the Layout, is rejected as a whole with nothing changed
- [ ] After the call each affected Card's Inventory Entry quantity equals its Slot count in that Collection
- [ ] Another user's binder, an unauthenticated caller and a plain Collection are rejected
- [ ] Covered by HTTP-boundary tests against real Postgres
