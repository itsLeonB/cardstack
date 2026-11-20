# 01: Name the Trade, label the sides, retire Invite

**What to build:** A participant opens a Trade and can tell at once which Trade it is, which side is theirs, and whether they can still invite someone. The page title follows the counterparty once they join and reads "New trade" while the seat is empty. The viewer's side reads "You". Invite appears only for the owner of an `open` Trade.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] The detail page title shows the counterparty's display name once someone has joined, and "New trade" while the Trade is `open`; the state stays visible as the page description.
- [ ] The viewer's side is labelled "You" whether the viewer holds the owner seat or the participant seat; the other side keeps its display name.
- [ ] The empty seat keeps its existing "waiting for someone to join" wording.
- [ ] No email address, holdings, private identifier or other participant data appears in the new labels.
- [ ] Invite is visible only to the owner while the state is `open`; it is absent in `negotiating`, `locked`, `executed` and `cancelled`, and absent for a participant who is not the owner.
- [ ] The existing Invite dialog, link text, Copy and native share behaviour are unchanged; no Collection or Wishlist publication control is added.
- [ ] A Guest and a signed-in non-participant still receive the same not-found answer, and Trade pages remain unindexed.
- [ ] An authenticated end-to-end case shows visible Trade content before asserting Invite is absent, with an `open`-owner positive case and a joined-owner negative case.
- [ ] The new labels and action state are checked on phone and desktop widths, in both themes, with visible focus and keyboard access.
- [ ] Focused checks and the full frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.
