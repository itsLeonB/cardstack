# 05: Replace Page operation

**What to build:** One authenticated backend operation replaces the whole contents of a Page. The request names the Page and carries one state per Slot in grid order: empty, a Card id, or Unidentified. It is idempotent on retry. A Card id the server does not know is stored as Unidentified, and the response reports that for that Slot. A Page that does not exist, or a request whose Slot count does not match the Layout, is rejected as a whole and changes nothing. The response gives a result per Slot (stored as sent, or stored Unidentified with the reason). The Inventory Entries of every Card the Page held before or holds after are updated in the same transaction so that they equal the Slot counts.

Spec: `.scratch/binder-scanning/spec.md` (Replace Page, Quantity invariant). The page-scan screen (ticket 06) is the first caller.

**Blocked by:** 03 (place, clear, replace and mark Unidentified)

**Status:** resolved

- [x] Replacing a Page sets each Slot to empty, the given Card, or Unidentified, and cards that were on the Page and are no longer there are removed from it
- [x] Sending the same request twice gives the same state and the same response
- [x] A Card id the server does not know is stored as Unidentified and reported for that Slot, and the other Slots are still applied
- [x] A Page number that does not exist, or a Slot count that does not match the Layout, is rejected as a whole with nothing changed
- [x] After the call each affected Card's Inventory Entry quantity equals its Slot count in that Collection
- [x] Another user's binder, an unauthenticated caller and a plain Collection are rejected
- [x] Covered by HTTP-boundary tests against real Postgres

## Answer

Implemented on `feat/binder-scanning` (backend). `PUT` on the Page replaces every Slot in one request (one state per Slot in grid order: empty, card, unidentified; max 36). Idempotent; an unknown or nil Card id is stored Unidentified with reason `unknown_card`; a wrong Slot count or a nonexistent Page rejects the whole request. Affected Cards' Inventory Entries are reconciled once per request at the end of the transaction. See `backend/openapi.json` for the exact path and payload.
